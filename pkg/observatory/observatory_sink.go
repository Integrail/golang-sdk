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

type ObservatorySink struct {
	baseUri string
	apiKey  string

	bufferSize int
	flushTimer *time.Timer
	flushDelay time.Duration
	buffer     []logger.Message
	mutex      sync.Mutex
}

func NewObservatorySink(baseUri string, apiKey string, bufferSize int, flushDelay time.Duration) *ObservatorySink {
	return &ObservatorySink{
		baseUri:    baseUri,
		apiKey:     apiKey,
		bufferSize: bufferSize,
		flushDelay: flushDelay,
	}
}

func (s *ObservatorySink) Write(msg logger.Message) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.buffer = append(s.buffer, msg)

	// Reset flush timer
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}
	s.flushTimer = time.AfterFunc(s.flushDelay, func() {
		s.Flush()
	})

	// Flush if buffer is full
	if len(s.buffer) >= s.bufferSize {
		return s.flush()
	}

	return nil
}

func (s *ObservatorySink) WriteImmediately(msgs []logger.Message) error {
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

	url := fmt.Sprintf("%s/api/v1/observatory/logs", s.baseUri)
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

func (s *ObservatorySink) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.flush()
}

func (s *ObservatorySink) flush() error {
	if len(s.buffer) == 0 {
		return nil
	}

	if err := s.WriteImmediately(s.buffer); err != nil {
		return fmt.Errorf("failed to write buffered messages: %w", err)
	}

	// Clear buffer
	s.buffer = s.buffer[:0]

	if s.flushTimer != nil {
		s.flushTimer.Stop()
		s.flushTimer = nil
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
