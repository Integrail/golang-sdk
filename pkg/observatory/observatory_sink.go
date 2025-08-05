package observatory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/simple-container-com/go-aws-lambda-sdk/pkg/logger"
)

type ObservatorySink struct {
	baseUri string
	apiKey  string
}

func NewObservatorySink(baseUri string, apiKey string) *ObservatorySink {
	return &ObservatorySink{
		baseUri: baseUri,
		apiKey:  apiKey,
	}
}

func (s *ObservatorySink) Write(msg logger.Message) error {
	var logLevel int
	switch msg.Level {
	case logger.Error:
		logLevel = 1
	case logger.Warn:
		logLevel = 2
	case logger.Info:
		logLevel = 3
	}
	body := ObservatoryPushLogsRequest{
		Logs: []ObservatoryPushLogsRequestLog{
			{
				Message:   msg.Message,
				LogLevel:  logLevel,
				Data:      msg.Context,
				Module:    nil,
				Submodule: nil,
			},
		},
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
