package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

type LogUploader struct {
	taskIdentification api.TaskIdentification
	commandName        string
	client             api.CirrusCIService_StreamLogsClient
	storedOutput       *os.File
	erroredChunks      int
	logsChannel        chan []byte
	doneLogUpload      chan bool
	valuesToMask       []string
	closed             bool
	mutex              sync.RWMutex
}

func NewLogUploader(executor *Executor, commandName string) (*LogUploader, error) {
	identifier := executor.taskIdentification
	logClient, err := InitializeLogStreamClient(identifier, commandName, false)
	if err != nil {
		return nil, err
	}
	EnsureFolderExists(os.TempDir())
	file, err := ioutil.TempFile(os.TempDir(), commandName)
	if err != nil {
		return nil, err
	}
	logUploader := LogUploader{
		taskIdentification: identifier,
		commandName:        commandName,
		client:             logClient,
		storedOutput:       file,
		erroredChunks:      0,
		logsChannel:        make(chan []byte),
		doneLogUpload:      make(chan bool),
		valuesToMask:       executor.sensitiveValues,
		closed:             false,
	}
	go logUploader.StreamLogs()
	return &logUploader, nil
}

func (uploader *LogUploader) reInitializeClient() error {
	err := uploader.client.CloseSend()
	if err != nil {
		log.Printf("Failed to close log for %s for reinitialization: %s\n", uploader.commandName, err.Error())
	}
	logClient, err := InitializeLogStreamClient(uploader.taskIdentification, uploader.commandName, false)
	if err != nil {
		return err
	}
	uploader.client = logClient
	return nil
}

func (uploader *LogUploader) Write(bytes []byte) (int, error) {
	if len(bytes) == 0 {
		return 0, nil
	}
	uploader.mutex.RLock()
	defer uploader.mutex.RUnlock()
	if !uploader.closed {
		bytesCopy := make([]byte, len(bytes))
		copy(bytesCopy, bytes)
		uploader.logsChannel <- bytesCopy
	}
	return len(bytes), nil
}

func (uploader *LogUploader) StreamLogs() {
	for {
		logs, finished := uploader.ReadAvailableChunks()
		_, err := uploader.WriteChunk(logs)
		if finished {
			log.Printf("Finished streaming logs for %s!\n", uploader.commandName)
			break
		}
		if err == io.EOF {
			log.Printf("Got EOF while streaming logs for %s! Trying to reinitilize logs uploader...\n", uploader.commandName)
			err := uploader.reInitializeClient()
			if err == nil {
				log.Printf("Successfully reinitilized log uploader for %s!\n", uploader.commandName)
			} else {
				log.Printf("Failed to reinitilized log uploader for %s: %s\n", uploader.commandName, err.Error())
			}
		}
	}
	uploader.client.CloseAndRecv()

	err := uploader.UploadStoredOutput()
	if err != nil {
		log.Printf("Failed to upload stored logs for %s: %s", uploader.commandName, err.Error())
	} else {
		log.Printf("Uploaded stored logs for %s!", uploader.commandName)
	}

	uploader.storedOutput.Close()
	os.Remove(uploader.storedOutput.Name())

	uploader.doneLogUpload <- true
}

func (uploader *LogUploader) ReadAvailableChunks() ([]byte, bool) {
	result := <-uploader.logsChannel
	for {
		select {
		case nextChunk, more := <-uploader.logsChannel:
			result = append(result, nextChunk...)
			if !more {
				log.Printf("No more log chunks for %s\n", uploader.commandName)
				return result, true
			}
		default:
			return result, false
		}
	}
}

func (uploader *LogUploader) WriteChunk(bytesToWrite []byte) (int, error) {
	if len(bytesToWrite) == 0 {
		return 0, nil
	}
	for _, valueToMask := range uploader.valuesToMask {
		bytesToWrite = bytes.Replace(bytesToWrite, []byte(valueToMask), []byte("HIDDEN-BY-CIRRUS-CI"), -1)
	}

	uploader.storedOutput.Write(bytesToWrite)
	dataChunk := api.DataChunk{Data: bytesToWrite}
	logEntry := api.LogEntry_Chunk{Chunk: &dataChunk}
	err := uploader.client.Send(&api.LogEntry{Value: &logEntry})
	if err != nil {
		log.Printf("Failed to send logs! %s For %s", err.Error(), string(bytesToWrite))
		uploader.erroredChunks++
		return 0, err
	}
	return len(bytesToWrite), nil
}

func (uploader *LogUploader) Finalize() {
	log.Printf("Finilizing log uploading for %s!\n", uploader.commandName)
	uploader.mutex.Lock()
	uploader.closed = true
	close(uploader.logsChannel)
	uploader.mutex.Unlock()
	<-uploader.doneLogUpload
}

func (uploader *LogUploader) UploadStoredOutput() error {
	logClient, err := InitializeLogSaveClient(uploader.taskIdentification, uploader.commandName, true)
	if err != nil {
		return err
	}
	defer logClient.CloseAndRecv()

	if uploader.commandName == "test_unexpected_error_during_log_streaming" {
		dataChunk := api.DataChunk{Data: []byte("Live streaming of logs failed!\n")}
		logEntry := api.LogEntry_Chunk{Chunk: &dataChunk}
		err = logClient.Send(&api.LogEntry{Value: &logEntry})
		if err != nil {
			return err
		}
	}

	uploader.storedOutput.Seek(0, io.SeekStart)

	readBufferSize := int(1024 * 1024)
	readBuffer := make([]byte, readBufferSize)
	bufferedReader := bufio.NewReaderSize(uploader.storedOutput, readBufferSize)
	for {
		n, err := bufferedReader.Read(readBuffer)

		if n > 0 {
			dataChunk := api.DataChunk{Data: readBuffer[:n]}
			logEntry := api.LogEntry_Chunk{Chunk: &dataChunk}
			err = logClient.Send(&api.LogEntry{Value: &logEntry})
		}

		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func InitializeLogStreamClient(taskIdentification api.TaskIdentification, commandName string, raw bool) (api.CirrusCIService_StreamLogsClient, error) {
	streamLogClient, err := client.CirrusClient.StreamLogs(
		context.Background(),
		grpc.UseCompressor(gzip.Name),
	)
	if err != nil {
		time.Sleep(5 * time.Second)
		streamLogClient, err = client.CirrusClient.StreamLogs(
			context.Background(),
			grpc.UseCompressor(gzip.Name),
		)
	}
	if err != nil {
		time.Sleep(20 * time.Second)
		streamLogClient, err = client.CirrusClient.StreamLogs(
			context.Background(),
			grpc.UseCompressor(gzip.Name),
		)
	}
	if err != nil {
		log.Printf("Failed to start streaming logs for %s! %s", commandName, err.Error())
		request := api.ReportAgentProblemRequest{
			TaskIdentification: &taskIdentification,
			Message:            fmt.Sprintf("Failed to start streaming logs for command %v: %v", commandName, err),
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return nil, err
	}
	logEntryKey := api.LogEntry_LogKey{TaskIdentification: &taskIdentification, CommandName: commandName, Raw: raw}
	logEntry := api.LogEntry_Key{Key: &logEntryKey}
	streamLogClient.Send(&api.LogEntry{Value: &logEntry})
	return streamLogClient, nil
}

func InitializeLogSaveClient(taskIdentification api.TaskIdentification, commandName string, raw bool) (api.CirrusCIService_SaveLogsClient, error) {
	streamLogClient, err := client.CirrusClient.SaveLogs(
		context.Background(),
		grpc.UseCompressor(gzip.Name),
	)
	if err != nil {
		time.Sleep(5 * time.Second)
		streamLogClient, err = client.CirrusClient.SaveLogs(
			context.Background(),
			grpc.UseCompressor(gzip.Name),
		)
	}
	if err != nil {
		time.Sleep(20 * time.Second)
		streamLogClient, err = client.CirrusClient.SaveLogs(
			context.Background(),
			grpc.UseCompressor(gzip.Name),
		)
	}
	if err != nil {
		log.Printf("Failed to start saving logs for %s! %s", commandName, err.Error())
		request := api.ReportAgentProblemRequest{
			TaskIdentification: &taskIdentification,
			Message:            fmt.Sprintf("Failed to start saving logs for command %v: %v", commandName, err),
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return nil, err
	}
	logEntryKey := api.LogEntry_LogKey{TaskIdentification: &taskIdentification, CommandName: commandName, Raw: raw}
	logEntry := api.LogEntry_Key{Key: &logEntryKey}
	streamLogClient.Send(&api.LogEntry{Value: &logEntry})
	return streamLogClient, nil
}
