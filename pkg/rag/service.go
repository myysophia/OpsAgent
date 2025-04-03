package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ragAPIEndpoint = "https://dashscope.aliyuncs.com/api/v1/apps/%s/completion"
)

type RAGService struct {
	apiKey string
	appID  string
}

type RAGRequest struct {
	Input struct {
		Prompt string `json:"prompt"`
	} `json:"input"`
	Parameters map[string]interface{} `json:"parameters"`
	Debug      map[string]interface{} `json:"debug"`
}

type RAGResponse struct {
	Output struct {
		FinishReason string `json:"finish_reason"`
		SessionID    string `json:"session_id"`
		Text         string `json:"text"`
	} `json:"output"`
	Usage struct {
		Models []struct {
			OutputTokens int    `json:"output_tokens"`
			ModelID      string `json:"model_id"`
			InputTokens  int    `json:"input_tokens"`
		} `json:"models"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

type RecommendCommand struct {
	Recommend string `json:"recommend"`
}

func NewRAGService(apiKey, appID string) *RAGService {
	return &RAGService{
		apiKey: apiKey,
		appID:  appID,
	}
}

func (s *RAGService) GetRecommendCommand(query string) (string, error) {
	reqBody := RAGRequest{
		Input: struct {
			Prompt string `json:"prompt"`
		}{
			Prompt: query,
		},
		Parameters: map[string]interface{}{},
		Debug:      map[string]interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf(ragAPIEndpoint, s.appID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ragResp RAGResponse
	if err := json.Unmarshal(body, &ragResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 从text字段中提取JSON内容
	text := ragResp.Output.Text
	// 移除markdown代码块标记
	text = text[7 : len(text)-3] // 移除 ```json\n 和 \n```

	var recommendCmd RecommendCommand
	if err := json.Unmarshal([]byte(text), &recommendCmd); err != nil {
		return "", fmt.Errorf("failed to unmarshal recommend command: %w", err)
	}

	return recommendCmd.Recommend, nil
}
