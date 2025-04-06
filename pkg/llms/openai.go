/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package llms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient 封装了 OpenAI API 客户端
type OpenAIClient struct {
	*openai.Client

	Retries int           // 重试次数
	Backoff time.Duration // 重试间隔
}

// NewOpenAIClient 创建新的 OpenAI 客户端
// 支持标准 OpenAI API 和 Azure OpenAI API
func NewOpenAIClient(apiKey string, baseURL string) (*OpenAIClient, error) {
	//apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	config := openai.DefaultConfig(apiKey)
	//baseURL := os.Getenv("OPENAI_API_BASE")
	if baseURL != "" {
		config.BaseURL = baseURL

		if strings.Contains(baseURL, "azure") {
			config.APIType = openai.APITypeAzure
			config.APIVersion = "2024-06-01"
			config.AzureModelMapperFunc = func(model string) string {
				return regexp.MustCompile(`[.:]`).ReplaceAllString(model, "")
			}
		}

		// 为Qwen模型添加流式处理的请求头
		if strings.Contains(baseURL, "dashscope") {
			config.HTTPClient = &http.Client{
				Transport: &QwenTransport{},
			}
		}
	}

	return &OpenAIClient{
		Retries: 5,
		Backoff: time.Second,
		Client:  openai.NewClientWithConfig(config),
	}, nil
}

// QwenTransport 是一个自定义的HTTP传输层，用于为Qwen模型添加流式处理的请求头
type QwenTransport struct {
	http.Transport
}

// RoundTrip 实现http.RoundTripper接口，为请求添加Qwen流式处理的请求头
func (t *QwenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 为流式请求添加特定的请求头
	if strings.Contains(req.URL.String(), "dashscope") && req.Header.Get("Accept") == "text/event-stream" {
		req.Header.Set("X-DashScope-SSE", "enable")
	}
	return http.DefaultTransport.RoundTrip(req)
}

// Chat 执行与 LLM 的对话
// - model: 使用的模型名称
// - maxTokens: 最大 token 数量
// - prompts: 对话历史
func (c *OpenAIClient) Chat(model string, maxTokens int, prompts []openai.ChatCompletionMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: math.SmallestNonzeroFloat32,
		Messages:    prompts,
	}

	// 检查是否为Qwen模型，如果是则使用流式API
	if strings.Contains(strings.ToLower(model), "qwen") {
		return c.StreamChat(model, maxTokens, prompts)
	}

	backoff := c.Backoff
	for try := 0; try < c.Retries; try++ {
		resp, err := c.Client.CreateChatCompletion(context.Background(), req)

		if err == nil {
			return string(resp.Choices[0].Message.Content), nil
		}

		e := &openai.APIError{}

		if errors.As(err, &e) {
			switch e.HTTPStatusCode {
			case 401:
				return "", err
			case 429, 500:
				time.Sleep(backoff)
				backoff *= 2
				continue
			default:
				return "", err
			}
		}

		return "", err
	}

	return "", fmt.Errorf("OpenAI request throttled after retrying %d times", c.Retries)
}

// StreamChat 执行与 LLM 的流式对话
// - model: 使用的模型名称
// - maxTokens: 最大 token 数量
// - prompts: 对话历史
func (c *OpenAIClient) StreamChat(model string, maxTokens int, prompts []openai.ChatCompletionMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: math.SmallestNonzeroFloat32,
		Messages:    prompts,
		Stream:      true,
	}

	ctx := context.Background()
	backoff := c.Backoff

	for try := 0; try < c.Retries; try++ {
		stream, err := c.Client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			e := &openai.APIError{}
			if errors.As(err, &e) {
				switch e.HTTPStatusCode {
				case 401:
					return "", err
				case 429, 500:
					time.Sleep(backoff)
					backoff *= 2
					continue
				default:
					return "", err
				}
			}
			return "", err
		}
		defer stream.Close()

		var fullResponse strings.Builder
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", err
			}
			fullResponse.WriteString(response.Choices[0].Delta.Content)
		}

		return fullResponse.String(), nil
	}

	return "", fmt.Errorf("OpenAI request throttled after retrying %d times", c.Retries)
}
