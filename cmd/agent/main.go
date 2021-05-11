package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor"
	"github.com/cirruslabs/cirrus-ci-agent/internal/network"
	"github.com/cirruslabs/cirrus-ci-agent/pkg/grpchelper"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/keepalive"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
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
	preCreatedWorkingDir := flag.String("pre-created-working-dir", "",
		"working directory to use when spawned via Persistent Worker")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("cirrus-agent-%d.log", *taskIdPtr))
	if *stopHook {
		// In case of a failure the log file will be persisted on the machine for debugging purposes.
		// But unfortunately stop hook invocation will override it so let's use a different name.
		logFilePath = filepath.Join(os.TempDir(), fmt.Sprintf("cirrus-agent-%d-hook.log", *taskIdPtr))
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
	}
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(multiWriter, multiWriter, multiWriter))

	var conn *grpc.ClientConn

	_ = retry.Do(func() error {
		conn, err = dialWithTimeout(*apiEndpointPtr)
		return err
	}, retry.OnRetry(func(n uint, err error) {
		log.Printf("Failed to open a connection: %v\n", err)
	}), retry.Delay(1*time.Second), retry.MaxDelay(1*time.Second), retry.Attempts(math.MaxUint32))

	log.Printf("Connected!\n")
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
		} else {
			logFile.Close()
			os.Remove(logFilePath)
		}
		os.Exit(0)
	}

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered an error: %v", err)
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

	go runHeartbeat(*taskIdPtr, *clientTokenPtr, conn)

	buildExecutor := executor.NewExecutor(*taskIdPtr, *clientTokenPtr, *serverTokenPtr, *commandFromPtr, *commandToPtr,
		*preCreatedWorkingDir)
	buildExecutor.RunBuild()

	logFile.Close()
	uploadAgentLogs(logFilePath, *taskIdPtr, *clientTokenPtr)
}

func uploadAgentLogs(logFilePath string, taskId int64, clientToken string) {
	logContents, readErr := ioutil.ReadFile(logFilePath)
	if readErr != nil {
		return
	}
	taskIdentification := api.TaskIdentification{
		TaskId: taskId,
		Secret: clientToken,
	}
	request := api.ReportAgentLogsRequest{
		TaskIdentification: &taskIdentification,
		Logs:               string(logContents),
	}
	_, err := client.CirrusClient.ReportAgentLogs(context.Background(), &request)
	if err == nil {
		os.Remove(logFilePath)
	}
}

func dialWithTimeout(apiEndpoint string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	target, insecure := grpchelper.TransportSettings(apiEndpoint)

	// use embedded root certificates because the agent can be executed with a distroless container, for example
	// also don't check for error since then the default certificates from the host will be used
	certPool, _ := gocertifi.CACerts()
	tlsCredentials := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    certPool,
	})
	transportSecurity := grpc.WithTransportCredentials(tlsCredentials)

	if insecure {
		transportSecurity = grpc.WithInsecure()
	}
	retryCodes := []codes.Code{
		codes.Unavailable, codes.Internal, codes.Unknown, codes.ResourceExhausted, codes.DeadlineExceeded,
	}
	return grpc.DialContext(
		ctx,
		target,
		grpc.WithBlock(),
		transportSecurity,
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                30 * time.Second, // make connection is alive every 30 seconds
				Timeout:             60 * time.Second, // with a timeout of 60 seconds
				PermitWithoutStream: true,             // always send Pings even if there are no RPCs
			},
		),
		grpc.WithUnaryInterceptor(
			grpc_retry.UnaryClientInterceptor(
				grpc_retry.WithMax(3),
				grpc_retry.WithCodes(retryCodes...),
				grpc_retry.WithPerRetryTimeout(10*time.Second),
			),
		),
	)
}

func runHeartbeat(taskId int64, clientToken string, conn *grpc.ClientConn) {
	taskIdentification := api.TaskIdentification{
		TaskId: taskId,
		Secret: clientToken,
	}
	for {
		log.Println("Sending heartbeat...")
		_, err := client.CirrusClient.Heartbeat(context.Background(), &api.HeartbeatRequest{TaskIdentification: &taskIdentification})
		if err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
			connectionState := conn.GetState()
			log.Printf("Connection state: %v", connectionState.String())
			if connectionState == connectivity.TransientFailure {
				conn.ResetConnectBackoff()
			}
		} else {
			log.Printf("Sent heartbeat!")
		}
		time.Sleep(60 * time.Second)
	}
}
