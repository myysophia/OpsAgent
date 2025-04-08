package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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

// RecommendCommand 旧的推荐命令结构
type RecommendCommand struct {
	Recommend string `json:"context_name"`
}

// ContextKnowledge 新的上下文知识结构
type ContextKnowledge struct {
	ContextKnowledge []struct {
		ContextName string `json:"context_name"`
	} `json:"context_knowledge"`
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

	// 处理文本格式
	if strings.HasPrefix(text, "```json") && strings.HasSuffix(text, "```") {
		// 移除markdown代码块标记
		text = text[7 : len(text)-3] // 移除 ```json\n 和 \n```
	}

	// 尝试解析为新格式
	var contextKnowledge ContextKnowledge
	if err := json.Unmarshal([]byte(text), &contextKnowledge); err == nil {
		// 成功解析为新格式
		if len(contextKnowledge.ContextKnowledge) > 0 {
			contextName := contextKnowledge.ContextKnowledge[0].ContextName
			return contextName, nil
		}
	}

	// 尝试解析为旧格式
	var recommendCmd RecommendCommand
	if err := json.Unmarshal([]byte(text), &recommendCmd); err != nil {
		return "", fmt.Errorf("failed to unmarshal context knowledge: %w, text: %s", err, text)
	}

	// 如果旧格式解析成功，但值为空，尝试直接使用文本
	if recommendCmd.Recommend == "" {
		// 尝试直接从文本中提取上下文名称
		if strings.Contains(text, "context_name") {
			// 使用正则表达式提取上下文名称
			re := regexp.MustCompile(`"context_name"\s*:\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				contextName := matches[1]
				return contextName, nil
			}
		}
		return "", fmt.Errorf("failed to extract context name from text: %s", text)
	}

	return recommendCmd.Recommend, nil
}

// GetContextInfo 获取上下文信息
func (s *RAGService) GetContextInfo(query string) (string, error) {
	// 调用原来的推荐命令接口
	recommendCmd, err := s.GetRecommendCommand(query)
	if err != nil {
		return "", fmt.Errorf("failed to get recommend command: %w", err)
	}

	// 直接使用推荐命令作为上下文名称
	contextName := recommendCmd

	// 从上下文名称中提取区域 ID
	regionID := ""
	if strings.Contains(contextName, "-") {
		parts := strings.Split(contextName, "-")
		if len(parts) >= 2 {
			regionID = parts[len(parts)-1]
		}
	}

	// 构建新的上下文信息结构
	contextInfo := fmt.Sprintf(`{
  "kubecontext_knowledge": [
    {
      "context_name": "%s",
      "region_identifier": "%s"
    }
  ]
}`, contextName, regionID)

	return contextInfo, nil
}
