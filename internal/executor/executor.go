package executor

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-agent/internal/http_cache"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type CommandAndLogs struct {
	Name string
	Cmd  *exec.Cmd
	Logs *LogUploader
}

type Executor struct {
	taskIdentification *api.TaskIdentification
	serverToken        string
	backgroundCommands []CommandAndLogs
	httpCacheHost      string
	timeout            <-chan time.Time
	sensitiveValues    []string
	commandFrom        string
	commandTo          string
}

func NewExecutor(taskId int64, clientToken, serverToken string, commandFrom string, commandTo string) *Executor {
	taskIdentification := &api.TaskIdentification{
		TaskId: taskId,
		Secret: clientToken,
	}
	return &Executor{
		taskIdentification: taskIdentification,
		serverToken:        serverToken,
		backgroundCommands: make([]CommandAndLogs, 0),
		httpCacheHost:      "",
		sensitiveValues:    make([]string, 0),
		commandFrom:        commandFrom,
		commandTo:          commandTo,
	}
}

func (executor *Executor) RunBuild() {
	initialStepsRequest := api.InitialCommandsRequest{
		TaskIdentification:  executor.taskIdentification,
		LocalTimestamp:      time.Now().Unix(),
		ContinueFromCommand: executor.commandFrom,
	}
	log.Println("Getting initial commands...")
	response, err := client.CirrusClient.InitialCommands(context.Background(), &initialStepsRequest)
	for response == nil || err != nil {
		request := api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification,
			Message:            fmt.Sprintf("Failed to get initial commands: %v", err),
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		time.Sleep(10 * time.Second)
		response, err = client.CirrusClient.InitialCommands(context.Background(), &initialStepsRequest)
	}

	if response.ServerToken != executor.serverToken {
		log.Panic("Server token is incorrect!")
		return
	}

	environment := getExpandedScriptEnvironment(executor, response.Environment)

	workingDir, ok := environment["CIRRUS_WORKING_DIR"]
	if ok {
		EnsureFolderExists(workingDir)
		if err := os.Chdir(workingDir); err != nil {
			log.Printf("Failed to change current working directory to '%s': %v", workingDir, err)
		}
	} else {
		log.Printf("Not changing current working directory because CIRRUS_WORKING_DIR is not set")
	}

	commands := response.Commands

	if cacheHost, ok := os.LookupEnv("CIRRUS_HTTP_CACHE_HOST"); ok {
		environment["CIRRUS_HTTP_CACHE_HOST"] = cacheHost
	}

	if _, ok := environment["CIRRUS_HTTP_CACHE_HOST"]; !ok {
		environment["CIRRUS_HTTP_CACHE_HOST"] = http_cache.Start(executor.taskIdentification)
	}

	executor.httpCacheHost = environment["CIRRUS_HTTP_CACHE_HOST"]
	executor.timeout = time.After(time.Duration(response.TimeoutInSeconds) * time.Second)
	executor.sensitiveValues = response.SecretsToMask

	if len(commands) == 0 {
		return
	}

	failedAtLeastOnce := response.FailedAtLeastOnce

	for _, command := range BoundedCommands(commands, executor.commandFrom, executor.commandTo) {
		shouldRun := (command.ExecutionBehaviour == api.Command_ON_SUCCESS && !failedAtLeastOnce) ||
			(command.ExecutionBehaviour == api.Command_ON_FAILURE && failedAtLeastOnce) ||
			command.ExecutionBehaviour == api.Command_ALWAYS
		if !shouldRun {
			continue
		}

		log.Printf("Executing %s...", command.Name)
		if !executor.performStep(environment, command) {
			failedAtLeastOnce = true
		}
		log.Printf("%s finished!", command.Name)
	}
	log.Printf("Background commands to clean up after: %d!\n", len(executor.backgroundCommands))
	for i := 0; i < len(executor.backgroundCommands); i++ {
		backgroundCommand := executor.backgroundCommands[i]
		log.Printf("Cleaning up after background command %s...\n", backgroundCommand.Name)
		err := backgroundCommand.Cmd.Process.Kill()
		if err != nil {
			backgroundCommand.Logs.Write([]byte(fmt.Sprintf("\nFailed to stop background script %s: %s!", backgroundCommand.Name, err)))
		}
		backgroundCommand.Logs.Finalize()
	}
}

// BoundedCommands bounds a slice of commands with unique names to a half-open range [fromName, toName).
func BoundedCommands(commands []*api.Command, fromName, toName string) []*api.Command {
	left, right := 0, len(commands)

	for i, command := range commands {
		if fromName != "" && command.Name == fromName {
			left = i
		}

		if toName != "" && command.Name == toName {
			right = i
		}
	}

	return commands[left:right]
}

func getExpandedScriptEnvironment(executor *Executor, responseEnvironment map[string]string) map[string]string {
	if responseEnvironment == nil {
		responseEnvironment = make(map[string]string)
	}

	if _, ok := responseEnvironment["OS"]; !ok {
		if _, ok := os.LookupEnv("OS"); !ok {
			responseEnvironment["OS"] = runtime.GOOS
		}
	}
	responseEnvironment["CIRRUS_OS"] = runtime.GOOS

	if _, ok := responseEnvironment["CIRRUS_WORKING_DIR"]; !ok {
		defaultTempDirPath := filepath.Join(os.TempDir(), "cirrus-ci-build")
		if _, err := os.Stat(defaultTempDirPath); os.IsNotExist(err) {
			responseEnvironment["CIRRUS_WORKING_DIR"] = filepath.ToSlash(defaultTempDirPath)
		} else if executor.commandFrom != "" {
			// Default folder exists and we continue execution. Therefore we need to use it.
			responseEnvironment["CIRRUS_WORKING_DIR"] = filepath.ToSlash(defaultTempDirPath)
		} else {
			uniqueTempDirPath, _ := ioutil.TempDir(os.TempDir(), fmt.Sprintf("cirrus-task-%d", executor.taskIdentification.TaskId))
			responseEnvironment["CIRRUS_WORKING_DIR"] = filepath.ToSlash(uniqueTempDirPath)
		}
	}

	result := expandEnvironmentRecursively(responseEnvironment)

	return result
}

func (executor *Executor) performStep(env map[string]string, currentStep *api.Command) bool {
	success := false
	signaledToExit := false
	start := time.Now()

	switch instruction := currentStep.Instruction.(type) {
	case *api.Command_ExitInstruction:
		os.Exit(0)
	case *api.Command_CloneInstruction:
		success = executor.CloneRepository(env)
	case *api.Command_FileInstruction:
		success = executor.CreateFile(currentStep.Name, instruction.FileInstruction, env)
	case *api.Command_ScriptInstruction:
		cmd, err := executor.ExecuteScriptsStreamLogsAndWait(currentStep.Name, instruction.ScriptInstruction.Scripts, env)
		success = err == nil && cmd.ProcessState.Success()
		if err == nil {
			if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
				signaledToExit = ws.Signaled()
			}
		}
		if err == TimeOutError {
			signaledToExit = false
		}
	case *api.Command_BackgroundScriptInstruction:
		cmd, logClient, err := executor.ExecuteScriptsAndStreamLogs(currentStep.Name, instruction.BackgroundScriptInstruction.Scripts, env)
		if err == nil {
			executor.backgroundCommands = append(executor.backgroundCommands, CommandAndLogs{
				Name: currentStep.Name,
				Cmd:  cmd,
				Logs: logClient,
			})
			log.Printf("Started execution of #%d background command %s\n", len(executor.backgroundCommands), currentStep.Name)
			success = true
		} else {
			log.Printf("Failed to create command line for background command %s: %s\n", currentStep.Name, err)
			_, _ = logClient.Write([]byte(fmt.Sprintf("Failed to create command line: %s", err)))
			logClient.Finalize()
			success = false
		}
	case *api.Command_CacheInstruction:
		success = DownloadCache(executor, currentStep.Name, executor.httpCacheHost, instruction.CacheInstruction, env)
	case *api.Command_UploadCacheInstruction:
		success = UploadCache(executor, currentStep.Name, executor.httpCacheHost, instruction.UploadCacheInstruction, env)
	case *api.Command_ArtifactsInstruction:
		success = UploadArtifacts(executor, currentStep.Name, instruction.ArtifactsInstruction, env)
	default:
		log.Printf("Unsupported instruction %T", instruction)
		success = false
	}

	duration := time.Since(start)
	reportRequest := api.ReportSingleCommandRequest{
		TaskIdentification: executor.taskIdentification,
		CommandName:        (*currentStep).Name,
		Succeded:           success,
		DurationInSeconds:  int64(duration.Seconds()),
		SignaledToExit:     signaledToExit,
		LocalTimestamp:     time.Now().Unix(),
	}
	_, err := client.CirrusClient.ReportSingleCommand(context.Background(), &reportRequest)
	for err != nil {
		log.Printf("Failed to report command %v: %v\nRetrying...\n", (*currentStep).Name, err)
		time.Sleep(10 * time.Second)
		_, err = client.CirrusClient.ReportSingleCommand(context.Background(), &reportRequest)
	}
	return success
}

func (executor *Executor) ExecuteScriptsStreamLogsAndWait(
	commandName string,
	scripts []string,
	env map[string]string) (*exec.Cmd, error) {
	logUploader, err := NewLogUploader(executor, commandName, env)
	if err != nil {
		message := fmt.Sprintf("Failed to initialize command %v log upload: %v", commandName, err)
		request := api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification,
			Message:            message,
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return nil, errors.New(message)
	}
	defer logUploader.Finalize()
	cmd, err := ShellCommandsAndWait(scripts, &env, func(bytes []byte) (int, error) {
		return logUploader.Write(bytes)
	}, &executor.timeout)
	return cmd, err
}

func (executor *Executor) ExecuteScriptsAndStreamLogs(
	commandName string,
	scripts []string,
	env map[string]string) (*exec.Cmd, *LogUploader, error) {
	logUploader, err := NewLogUploader(executor, commandName, env)
	if err != nil {
		message := fmt.Sprintf("Failed to initialize command %v log upload: %v", commandName, err)
		request := api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification,
			Message:            message,
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return nil, logUploader, errors.New(message)
	}
	cmd, err := ShellCommands(scripts, &env, func(bytes []byte) (int, error) {
		return logUploader.Write(bytes)
	})
	return cmd, logUploader, err
}

func (executor *Executor) CreateFile(
	commandName string,
	instruction *api.FileInstruction,
	env map[string]string,
) bool {
	logUploader, err := NewLogUploader(executor, commandName, env)
	if err != nil {
		request := api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification,
			Message:            fmt.Sprintf("Failed to initialize command clone log upload: %v", err),
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return false
	}
	defer logUploader.Finalize()

	switch source := instruction.GetSource().(type) {
	case *api.FileInstruction_FromEnvironmentVariable:
		envName := source.FromEnvironmentVariable
		content, is_provided := env[envName]
		if !is_provided {
			logUploader.Write([]byte(fmt.Sprintf("Environment variable %s is not set! Skipping file creation...", envName)))
			return true
		}
		if strings.HasPrefix(content, "ENCRYPTED") {
			logUploader.Write([]byte(fmt.Sprintf("Environment variable %s wasn't decrypted! Skipping file creation...", envName)))
			return true
		}
		filePath := ExpandText(instruction.DestinationPath, env)
		EnsureFolderExists(filepath.Dir(filePath))
		err := ioutil.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("Failed to write file %s: %s!", filePath, err)))
			return false
		}
		logUploader.Write([]byte(fmt.Sprintf("Created file %s!", filePath)))
		return true
	default:
		log.Printf("Unsupported source %T", source)
		return false
	}
}

func (executor *Executor) CloneRepository(env map[string]string) bool {
	logUploader, err := NewLogUploader(executor, "clone", env)
	if err != nil {
		request := api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification,
			Message:            fmt.Sprintf("Failed to initialize command clone log upload: %v", err),
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return false
	}
	defer logUploader.Finalize()

	logUploader.Write([]byte("Using built-in Git...\n"))

	working_dir := env["CIRRUS_WORKING_DIR"]
	change := env["CIRRUS_CHANGE_IN_REPO"]
	branch := env["CIRRUS_BRANCH"]
	pr_number, is_pr := env["CIRRUS_PR"]
	tag, is_tag := env["CIRRUS_TAG"]
	_, is_clone_modules := env["CIRRUS_CLONE_SUBMODULES"]

	clone_url := env["CIRRUS_REPO_CLONE_URL"]
	if _, has_clone_token := env["CIRRUS_REPO_CLONE_TOKEN"]; has_clone_token {
		clone_url = ExpandText("https://x-access-token:${CIRRUS_REPO_CLONE_TOKEN}@${CIRRUS_REPO_CLONE_HOST}/${CIRRUS_REPO_FULL_NAME}.git", env)
	}

	clone_depth := 0
	if depth_str, ok := env["CIRRUS_CLONE_DEPTH"]; ok {
		clone_depth, _ = strconv.Atoi(depth_str)
	}
	if clone_depth > 0 {
		logUploader.Write([]byte(fmt.Sprintf("\nLimiting clone depth to %d!", clone_depth)))
	}

	// if an environment doesn't have git installed most likely it an alpine container
	// which also most likely doesn't have CA certificates so SSL will fail :-(
	// let's configure CA certs our self!
	cert_pool, err := gocertifi.CACerts()
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to get CA certificates: %s!", err)))
		return false
	}
	customClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: cert_pool},
		},
		Timeout: 900 * time.Second,
	}
	gitclient.InstallProtocol("https", githttp.NewClient(customClient))
	gitclient.InstallProtocol("http", githttp.NewClient(customClient))

	var repo *git.Repository

	if is_pr {
		repo, err = git.PlainInit(working_dir, false)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to init repository: %s!", err)))
			return false
		}
		remoteConfig := &config.RemoteConfig{
			Name: "origin",
			URLs: []string{clone_url},
		}
		if _, err := repo.CreateRemote(remoteConfig); err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to create remote: %s!", err)))
			return false
		}

		refSpec := fmt.Sprintf("+refs/pull/%s/head:refs/remotes/origin/pull/%[1]s", pr_number)
		logUploader.Write([]byte(fmt.Sprintf("\nFetching %s...\n", refSpec)))
		fetchOptions := &git.FetchOptions{
			RemoteName: remoteConfig.Name,
			RefSpecs:   []config.RefSpec{config.RefSpec(refSpec)},
			Tags:       git.NoTags,
			Progress:   logUploader,
		}
		if clone_depth > 0 {
			fetchOptions.Depth = clone_depth
		}
		err = repo.Fetch(fetchOptions)
		if err != nil && retryableCloneError(err) {
			logUploader.Write([]byte(fmt.Sprintf("\nFetch failed: %s!", err)))
			logUploader.Write([]byte("\nRe-trying to fetch..."))
			err = repo.Fetch(fetchOptions)
		}
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed fetch: %s!", err)))
			return false
		}

		workTree, err := repo.Worktree()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get work tree: %s!", err)))
			return false
		}

		checkoutOptions := git.CheckoutOptions{
			Hash: plumbing.NewHash(change),
		}
		logUploader.Write([]byte(fmt.Sprintf("\nChecking out %s...", checkoutOptions.Hash)))
		err = workTree.Checkout(&checkoutOptions)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to checkout %s: %s!", checkoutOptions.Hash, err)))
			return false
		}
	} else {
		cloneOptions := git.CloneOptions{
			URL:      clone_url,
			Progress: logUploader,
		}
		if clone_depth > 0 {
			cloneOptions.Depth = clone_depth
		}
		if !is_tag {
			cloneOptions.Tags = git.NoTags
		}

		if is_tag {
			cloneOptions.SingleBranch = true
			cloneOptions.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", tag))
		} else {
			cloneOptions.SingleBranch = true
			cloneOptions.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
		}
		logUploader.Write([]byte(fmt.Sprintf("\nCloning %s...\n", cloneOptions.ReferenceName)))

		repo, err = git.PlainClone(working_dir, false, &cloneOptions)

		if err != nil && retryableCloneError(err) {
			logUploader.Write([]byte(fmt.Sprintf("\nRetryable error '%s' while cloning! Trying again...", err)))
			os.RemoveAll(working_dir)
			EnsureFolderExists(working_dir)
			repo, err = git.PlainClone(working_dir, false, &cloneOptions)
		}

		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "timeout") || strings.Contains(strings.ToLower(err.Error()), "timed out") {
				logUploader.Write([]byte("\nFailed to clone because of a timeout from Git server!"))
			} else {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to clone: %s!", err)))
			}
			return false
		}
	}

	ref, err := repo.Head()
	if err != nil {
		logUploader.Write([]byte("\nFailed to get HEAD information!"))
		return false
	}

	if is_clone_modules {
		logUploader.Write([]byte("\nUpdating submodules..."))

		workTree, err := repo.Worktree()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get work tree: %s!", err)))
			return false
		}

		submodules, err := workTree.Submodules()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get submodules: %s!", err)))
			return false
		}

		if err := submodules.Update(&git.SubmoduleUpdateOptions{
			Init:              true,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		}); err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to update submodules: %s!", err)))
			return false
		}

		logUploader.Write([]byte("\nSucessfully updated submodules!"))
	}

	if ref.Hash() != plumbing.NewHash(change) {
		logUploader.Write([]byte(fmt.Sprintf("\nHEAD is at %s.", ref.Hash())))
		logUploader.Write([]byte(fmt.Sprintf("\nHard resetting to %s...", change)))

		workTree, err := repo.Worktree()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get work tree: %s!", err)))
			return false
		}

		err = workTree.Reset(&git.ResetOptions{
			Commit: plumbing.NewHash(change),
			Mode:   git.HardReset,
		})
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to force reset to %s: %s!", change, err)))
			return false
		}
	}
	logUploader.Write([]byte(fmt.Sprintf("\nChecked out %s on %s branch.", change, branch)))
	logUploader.Write([]byte("\nSuccessfully cloned!"))
	return true
}

func retryableCloneError(err error) bool {
	if err == nil {
		return false
	}
	errorMessage := strings.ToLower(err.Error())
	if strings.Contains(errorMessage, "timeout") {
		return true
	}
	if strings.Contains(errorMessage, "tls") {
		return true
	}
	if strings.Contains(errorMessage, "connection") {
		return true
	}
	if strings.Contains(errorMessage, "authentication") {
		return true
	}
	if strings.Contains(errorMessage, "not found") {
		return true
	}
	return false
}
