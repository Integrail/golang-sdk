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
}

func NewObservatorySink(baseUri string) *ObservatorySink {
	return &ObservatorySink{
		baseUri: baseUri,
	}
}

func (s *ObservatorySink) Write(msg logger.Message) error {
	body := ObservatoryPushLogsRequest{
		Logs: []ObservatoryPushLogsRequestLog{
			{
				Message:   msg.Message,
				LogLevel:  msg.Level,
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
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
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
	LogLevel  string  `json:"logLevel"`
	Data      any     `json:"data"`
	Module    *string `json:"module"`
	Submodule *string `json:"submodule"`
}
