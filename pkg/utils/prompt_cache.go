package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	promptCacheDir = ".cache/prompts"
	promptFileName = "execute_system_prompt.json"
	cacheDuration  = 24 * time.Hour
)

type PromptCache struct {
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	promptCache     *PromptCache
	promptCacheLock sync.RWMutex
	logger          = GetLogger()
)

// GetSystemPrompt 获取系统 prompt，优先从缓存获取，缓存不存在或过期则从 OSS 获取
func GetSystemPrompt() (string, error) {
	promptCacheLock.RLock()
	if promptCache != nil && time.Since(promptCache.UpdatedAt) < cacheDuration {
		content := promptCache.Content
		promptCacheLock.RUnlock()
		return content, nil
	}
	promptCacheLock.RUnlock()

	// 尝试从本地缓存文件读取
	if content, err := readFromCache(); err == nil {
		promptCacheLock.Lock()
		promptCache = &PromptCache{
			Content:   content,
			UpdatedAt: time.Now(),
		}
		promptCacheLock.Unlock()
		return content, nil
	}

	// 从 OSS 获取
	content, err := fetchFromOSS()
	if err != nil {
		return "", fmt.Errorf("failed to fetch prompt from OSS: %w", err)
	}

	// 更新缓存
	promptCacheLock.Lock()
	promptCache = &PromptCache{
		Content:   content,
		UpdatedAt: time.Now(),
	}
	promptCacheLock.Unlock()

	// 保存到本地缓存文件
	if err := saveToCache(content); err != nil {
		logger.Warn("failed to save prompt to cache file", zap.Error(err))
	}

	return content, nil
}

// readFromCache 从本地缓存文件读取 prompt
func readFromCache() (string, error) {
	cachePath := filepath.Join(promptCacheDir, promptFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", err
	}

	var cache PromptCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return "", err
	}

	if time.Since(cache.UpdatedAt) >= cacheDuration {
		return "", fmt.Errorf("cache expired")
	}

	return cache.Content, nil
}

// saveToCache 保存 prompt 到本地缓存文件
func saveToCache(content string) error {
	if err := os.MkdirAll(promptCacheDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(promptCacheDir, promptFileName)
	cache := PromptCache{
		Content:   content,
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// fetchFromOSS 从 OSS 获取 prompt
func fetchFromOSS() (string, error) {
	ossURL := "https://pub-10375556b89a45e0a56aff68854a2214.r2.dev/demo-prompt.md"

	resp, err := http.Get(ossURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch prompt from OSS: status code %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
