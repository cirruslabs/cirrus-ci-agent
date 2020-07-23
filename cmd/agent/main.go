package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor"
	"github.com/cirruslabs/cirrus-ci-agent/internal/network"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func main() {
	apiEndpointPtr := flag.String("api-endpoint", "https://grpc.cirrus-ci.com:443", "GRPC endpoint URL")
	taskIdPtr := flag.Int64("task-id", 0, "Task ID")
	clientTokenPtr := flag.String("client-token", "", "Secret token")
	serverTokenPtr := flag.String("server-token", "", "Secret token")
	help := flag.Bool("help", false, "help flag")
	stopHook := flag.Bool("stop-hook", false, "pre stop flag")
	commandFromPtr := flag.String("command-from", "", "Command to star execution from (inclusive)")
	commandToPtr := flag.String("command-to", "", "Command to stop execution at (exclusive)")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile, err := os.OpenFile("cirrus-agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
	}
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(multiWriter, multiWriter, multiWriter))

	var conn *grpc.ClientConn
	for {
		newConnection, err := dialWithTimeout(*apiEndpointPtr)
		if err == nil {
			conn = newConnection
			log.Printf("Connected!\n")
			break
		}
		log.Printf("Failed to open a connection: %v\n", err)
		time.Sleep(1 * time.Second)
	}
	defer conn.Close()

	client.InitClient(conn)

	if *stopHook {
		log.Printf("Stop hook!\n")
		taskIdentification := api.TaskIdentification{
			TaskId: *taskIdPtr,
			Secret: *clientTokenPtr,
		}
		request := api.ReportStopHookRequest{
			TaskIdentification: &taskIdentification,
		}
		_, err = client.CirrusClient.ReportStopHook(context.Background(), &request)
		if err != nil {
			log.Printf("Failed to report stop hook for task %d: %v\n", *taskIdPtr, err)
		}
		os.Exit(0)
	}

	defer func() {
		if err := recover(); err != nil {
			taskIdentification := api.TaskIdentification{
				TaskId: *taskIdPtr,
				Secret: *clientTokenPtr,
			}
			request := api.ReportAgentProblemRequest{
				TaskIdentification: &taskIdentification,
				Message:            fmt.Sprint(err),
				Stack:              string(debug.Stack()),
			}
			_, _ = client.CirrusClient.ReportAgentError(context.Background(), &request)
		}
	}()

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel)
	go func() {
		sig := <-signalChannel
		log.Printf("Captured %v...", sig)
		taskIdentification := api.TaskIdentification{
			TaskId: *taskIdPtr,
			Secret: *clientTokenPtr,
		}
		request := api.ReportAgentSignalRequest{
			TaskIdentification: &taskIdentification,
			Signal:             sig.String(),
		}
		_, _ = client.CirrusClient.ReportAgentSignal(context.Background(), &request)
	}()

	if portsToWait, ok := os.LookupEnv("CIRRUS_PORTS_WAIT_FOR"); ok {
		ports := strings.Split(portsToWait, ",")

		for _, port := range ports {
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				continue
			}
			log.Printf("Waiting on port %v...\n", port)
			network.WaitForLocalPort(portNumber, 60*time.Second)
		}
	}

	startHeartbeat(*taskIdPtr, *clientTokenPtr)

	buildExecutor := executor.NewExecutor(*taskIdPtr, *clientTokenPtr, *serverTokenPtr, *commandFromPtr, *commandToPtr)
	buildExecutor.RunBuild()
	err = logFile.Close()
	if err == nil {
		logFile, err = os.Open("cirrus-agent.log")
		logContents, readErr := ioutil.ReadAll(logFile)
		if err == nil && readErr == nil {
			taskIdentification := api.TaskIdentification{
				TaskId: *taskIdPtr,
				Secret: *clientTokenPtr,
			}
			request := api.ReportAgentLogsRequest{
				TaskIdentification: &taskIdentification,
				Logs:               string(logContents),
			}
			_, _ = client.CirrusClient.ReportAgentLogs(context.Background(), &request)
		}
	}
}

func dialWithTimeout(apiEndpoint string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	target, insecure := transportSettings(apiEndpoint)

	certPool, _ := gocertifi.CACerts()
	tlsCredentials := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    certPool,
	})
	transportSecurity := grpc.WithTransportCredentials(tlsCredentials)

	if insecure {
		transportSecurity = grpc.WithInsecure()
	}
	return grpc.DialContext(
		ctx,
		target,
		grpc.WithBlock(),
		transportSecurity,
		grpc.WithUnaryInterceptor(
			grpc_retry.UnaryClientInterceptor(
				grpc_retry.WithMax(3),
			),
		),
	)
}

func transportSettings(apiEndpoint string) (string, bool) {
	// Insecure by default to preserve backwards compatibility
	insecure := true

	// Use TLS if explicitly asked or no schema is in the target
	if strings.Contains(apiEndpoint, "https://") || !strings.Contains(apiEndpoint, "://") {
		insecure = false
	}
	// sanitize but leave unix:// if presented
	target := strings.TrimPrefix(strings.TrimPrefix(apiEndpoint, "http://"), "https://")
	return target, insecure
}

func startHeartbeat(taskId int64, clientToken string) {
	sendHeartbeat(taskId, clientToken)
	go heartbeat(taskId, clientToken)
}

func heartbeat(taskId int64, clientToken string) {
	ticker := time.NewTicker(60 * time.Second)
	for {
		sendHeartbeat(taskId, clientToken)
		<-ticker.C
	}
}

func sendHeartbeat(taskId int64, clientToken string) {
	taskIdentification := api.TaskIdentification{
		TaskId: taskId,
		Secret: clientToken,
	}
	_, err := client.CirrusClient.Heartbeat(context.Background(), &api.HeartbeatRequest{TaskIdentification: &taskIdentification})
	if err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
	} else {
		log.Printf("Sent heartbeat!")
	}
}
