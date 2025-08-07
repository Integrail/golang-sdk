package observatory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/simple-container-com/go-aws-lambda-sdk/pkg/logger"
)

type ObservatorySinkBuffer struct {
	messages []logger.Message
	mutex    sync.Mutex
}

type ObservatorySink struct {
	apiKey string

	bufferSize int
	flushTimer *time.Timer
	flushDelay time.Duration
	buffers    map[string]*ObservatorySinkBuffer
	mutex      sync.Mutex
}

func NewObservatorySink(apiKey string, bufferSize int, flushDelay time.Duration) *ObservatorySink {
	s := &ObservatorySink{
		apiKey:     apiKey,
		bufferSize: bufferSize,
		flushDelay: flushDelay,
		buffers:    make(map[string]*ObservatorySinkBuffer),
	}

	ticker := time.NewTicker(flushDelay)
	go func() {
		for range ticker.C {
			s.flush()
		}
	}()

	return s
}

func (s *ObservatorySink) Write(msg logger.Message) error {
	ctxValue := logger.ContextValue(msg.Context)
	everworkerCtx := ctxValue["executionCtx"]
	if everworkerCtx == nil {
		return nil
	}
	everworkerUrl := everworkerCtx.(map[string]any)["everworkerUrl"]
	if everworkerUrl == nil {
		return nil
	}

	s.mutex.Lock()
	if s.buffers[everworkerUrl.(string)] == nil {
		s.buffers[everworkerUrl.(string)] = &ObservatorySinkBuffer{
			messages: make([]logger.Message, 0, s.bufferSize),
		}
	}
	s.mutex.Unlock()

	buffer := s.buffers[everworkerUrl.(string)]
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	buffer.messages = append(buffer.messages, msg)

	return nil
}

func (s *ObservatorySink) writeImmediately(everworkerUrl string, msgs []logger.Message) error {
	body := ObservatoryPushLogsRequest{
		Logs: make([]ObservatoryPushLogsRequestLog, len(msgs)),
	}
	for i, msg := range msgs {
		var logLevel int
		switch msg.Level {
		case logger.Error:
			logLevel = 1
		case logger.Warn:
			logLevel = 2
		case logger.Info:
			logLevel = 3
		}
		body.Logs[i] = ObservatoryPushLogsRequestLog{
			Message:   msg.Message,
			LogLevel:  logLevel,
			Data:      msg.Context,
			Module:    nil,
			Submodule: nil,
		}
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal observatory log message: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/observatory/logs", everworkerUrl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("Error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send observatory log message: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("observatory log request failed with status: %s", resp.Status)
	}
	return nil
}

func (s *ObservatorySink) flush() error {
	for key, buffer := range s.buffers {
		buffer.mutex.Lock()
		defer buffer.mutex.Unlock()

		if len(buffer.messages) == 0 {
			return nil
		}

		if err := s.writeImmediately(key, buffer.messages); err != nil {
			return fmt.Errorf("failed to write buffered messages to %s: %w", key, err)
		}

		// Clear buffer
		buffer.messages = buffer.messages[:0]
	}

	return nil
}

type ObservatoryPushLogsRequest struct {
	Logs []ObservatoryPushLogsRequestLog `json:"logs"`
}

type ObservatoryPushLogsRequestLog struct {
	Message   string  `json:"message"`
	LogLevel  int     `json:"logLevel"`
	Data      any     `json:"data"`
	Module    *string `json:"module"`
	Submodule *string `json:"submodule"`
}
