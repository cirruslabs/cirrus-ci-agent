package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor"
	"github.com/cirruslabs/cirrus-ci-agent/internal/network"
	"github.com/cirruslabs/cirrus-ci-agent/internal/signalfilter"
	"github.com/cirruslabs/cirrus-ci-agent/pkg/grpchelper"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	goversion "github.com/hashicorp/go-version"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
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
	"syscall"
	"time"
)

var (
	version = "unknown"
	commit  = "unknown"
)

func fullVersion() string {
	var versionToNormalize string

	if version == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok {
			versionToNormalize = info.Main.Version
		}
	} else {
		versionToNormalize = version
	}

	// We parse the version here for two reasons:
	// * to weed out the "(devel)" version and fallback to "unknown" instead
	//   (see https://github.com/golang/go/issues/29228 for details on when this might happen)
	// * to remove the "v" prefix from the BuildInfo's version (e.g. "v0.7.0") and thus be consistent
	//   with the binary builds, where the version string would be "0.7.0" instead
	semver, err := goversion.NewSemver(versionToNormalize)
	if err == nil {
		version = semver.String()
	}

	return fmt.Sprintf("%s-%s", version, commit)
}

func main() {
	apiEndpointPtr := flag.String("api-endpoint", "https://grpc.cirrus-ci.com:443", "GRPC endpoint URL")
	taskIdPtr := flag.Int64("task-id", 0, "Task ID")
	clientTokenPtr := flag.String("client-token", "", "Secret token")
	serverTokenPtr := flag.String("server-token", "", "Secret token")
	versionFlag := flag.Bool("version", false, "display the version and exit")
	help := flag.Bool("help", false, "help flag")
	stopHook := flag.Bool("stop-hook", false, "pre stop flag")
	commandFromPtr := flag.String("command-from", "", "Command to star execution from (inclusive)")
	commandToPtr := flag.String("command-to", "", "Command to stop execution at (exclusive)")
	preCreatedWorkingDir := flag.String("pre-created-working-dir", "",
		"working directory to use when spawned via Persistent Worker")
	flag.Parse()

	if *versionFlag {
		fmt.Println(fullVersion())
		os.Exit(0)
	}

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	var conn *grpc.ClientConn

	logFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("cirrus-agent-%d.log", *taskIdPtr))
	if *stopHook {
		// In case of a failure the log file will be persisted on the machine for debugging purposes.
		// But unfortunately stop hook invocation will override it so let's use a different name.
		logFilePath = filepath.Join(os.TempDir(), fmt.Sprintf("cirrus-agent-%d-hook.log", *taskIdPtr))
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
	} else {
		defer func() {
			_ = logFile.Close()
			uploadAgentLogs(context.Background(), logFilePath, *taskIdPtr, *clientTokenPtr)
			if conn != nil {
				conn.Close()
			}
		}()
	}
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(multiWriter, multiWriter, multiWriter))

	log.Printf("Running agent version %s", fullVersion())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel)
	go func() {
		limiter := rate.NewLimiter(1, 1)

		for {
			sig := <-signalChannel

			if sig == os.Interrupt || sig == syscall.SIGTERM {
				cancel()
			}

			if signalfilter.IsNoisy(sig) || !limiter.Allow() {
				continue
			}

			log.Printf("Captured %v...", sig)

			reportSignal(context.Background(), sig, *taskIdPtr, *clientTokenPtr)
		}
	}()

	err = retry.Do(
		func() error {
			conn, err = dialWithTimeout(ctx, *apiEndpointPtr)
			return err
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("Failed to open a connection: %v\n", err)
		}),
		retry.Delay(1*time.Second), retry.MaxDelay(1*time.Second),
		retry.Attempts(math.MaxUint32), retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	if err != nil {
		// Context was cancelled before we had a chance to connect
		return
	}

	log.Printf("Connected!\n")

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
		_, err = client.CirrusClient.ReportStopHook(ctx, &request)
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

	if portsToWait, ok := os.LookupEnv("CIRRUS_PORTS_WAIT_FOR"); ok {
		ports := strings.Split(portsToWait, ",")

		for _, port := range ports {
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				continue
			}

			log.Printf("Waiting on port %v...\n", port)

			subCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			portErr := network.WaitForLocalPort(subCtx, portNumber)'
			if portErr != nil {
				log.Printf("Failed to wait fo port %v: %v\n", port, portErr)
			}
			cancel()
		}
	}

	go runHeartbeat(*taskIdPtr, *clientTokenPtr, conn)

	buildExecutor := executor.NewExecutor(*taskIdPtr, *clientTokenPtr, *serverTokenPtr, *commandFromPtr, *commandToPtr,
		*preCreatedWorkingDir)
	buildExecutor.RunBuild(ctx)
}

func uploadAgentLogs(ctx context.Context, logFilePath string, taskId int64, clientToken string) {
	if client.CirrusClient == nil {
		return
	}

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
	_, err := client.CirrusClient.ReportAgentLogs(ctx, &request)
	if err == nil {
		os.Remove(logFilePath)
	}
}

func reportSignal(ctx context.Context, sig os.Signal, taskId int64, clientToken string) {
	if client.CirrusClient == nil {
		return
	}

	taskIdentification := api.TaskIdentification{
		TaskId: taskId,
		Secret: clientToken,
	}
	request := api.ReportAgentSignalRequest{
		TaskIdentification: &taskIdentification,
		Signal:             sig.String(),
	}
	_, _ = client.CirrusClient.ReportAgentSignal(ctx, &request)
}

func dialWithTimeout(ctx context.Context, apiEndpoint string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	target, transportSecurity := grpchelper.TransportSettingsAsDialOption(apiEndpoint)

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
				grpc_retry.WithPerRetryTimeout(60*time.Second),
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
