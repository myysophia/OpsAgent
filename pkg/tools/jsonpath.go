package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// 处理 jsonpath 表达式
func processJSONPath(data []byte, jsonpath string) (string, error) {
	// 解析 JSON 数据
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return "", fmt.Errorf("解析 JSON 失败: %v", err)
	}

	// 如果是对象，转换为 map
	jsonMap, ok := jsonData.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("JSON 数据不是对象格式")
	}

	// 处理 items 数组
	items, ok := jsonMap["items"].([]interface{})
	if !ok {
		return "", fmt.Errorf("未找到 items 数组")
	}

	var results []string
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// 获取命名空间
		namespace, _ := itemMap["metadata"].(map[string]interface{})["namespace"].(string)
		name, _ := itemMap["metadata"].(map[string]interface{})["name"].(string)

		// 获取容器镜像
		var images []string
		if spec, ok := itemMap["spec"].(map[string]interface{}); ok {
			if containers, ok := spec["containers"].([]interface{}); ok {
				for _, container := range containers {
					if containerMap, ok := container.(map[string]interface{}); ok {
						if image, ok := containerMap["image"].(string); ok {
							images = append(images, image)
						}
					}
				}
			}
		}

		// 组合结果
		result := fmt.Sprintf("%s %s %s", namespace, name, strings.Join(images, " "))
		results = append(results, result)
	}

	return strings.Join(results, "\n"), nil
} 