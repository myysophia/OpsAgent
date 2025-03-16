package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// CleanJSON 清理非标准JSON字符串，使其符合标准格式
// 参数：
//   - jsonStr: 可能包含非标准格式的JSON字符串
//
// 返回：
//   - string: 清理后的标准JSON字符串
func CleanJSON(jsonStr string) string {
	// 移除可能的前缀和后缀非JSON内容
	jsonStr = extractJSONObject(jsonStr)

	// 处理多行字符串中的换行符
	jsonStr = handleMultilineStrings(jsonStr)

	// 处理未转义的引号
	jsonStr = handleUnescapedQuotes(jsonStr)

	// 处理尾部逗号
	jsonStr = handleTrailingCommas(jsonStr)

	return jsonStr
}

// extractJSONObject 从文本中提取JSON对象
// 参数：
//   - text: 可能包含JSON对象的文本
//
// 返回：
//   - string: 提取的JSON对象字符串
func extractJSONObject(text string) string {
	// 查找第一个左大括号和最后一个右大括号
	firstBrace := strings.Index(text, "{")
	lastBrace := strings.LastIndex(text, "}")

	if firstBrace == -1 || lastBrace == -1 || firstBrace > lastBrace {
		return text // 未找到有效的JSON对象
	}

	return text[firstBrace : lastBrace+1]
}

// handleMultilineStrings 处理多行字符串中的换行符
// 参数：
//   - jsonStr: JSON字符串
//
// 返回：
//   - string: 处理后的JSON字符串
func handleMultilineStrings(jsonStr string) string {
	// 在字符串值中将实际换行符替换为\n
	inString := false
	escaped := false
	var result strings.Builder

	for _, char := range jsonStr {
		switch char {
		case '\\':
			escaped = !escaped
			result.WriteRune(char)
		case '"':
			if !escaped {
				inString = !inString
			}
			escaped = false
			result.WriteRune(char)
		case '\n', '\r':
			if inString {
				if char == '\n' {
					result.WriteString("\\n")
				} else if char == '\r' {
					result.WriteString("\\r")
				}
			} else {
				result.WriteRune(char)
			}
			escaped = false
		default:
			escaped = false
			result.WriteRune(char)
		}
	}

	return result.String()
}

// handleUnescapedQuotes 处理未转义的引号
// 参数：
//   - jsonStr: JSON字符串
//
// 返回：
//   - string: 处理后的JSON字符串
func handleUnescapedQuotes(jsonStr string) string {
	// 使用正则表达式查找字符串值中未转义的引号
	re := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"`)
	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// 转义字符串值中的引号
		inner := match[1 : len(match)-1]
		inner = strings.ReplaceAll(inner, `"`, `\"`)
		return `"` + inner + `"`
	})
}

// handleTrailingCommas 处理尾部逗号
// 参数：
//   - jsonStr: JSON字符串
//
// 返回：
//   - string: 处理后的JSON字符串
func handleTrailingCommas(jsonStr string) string {
	// 移除对象和数组中的尾部逗号
	re := regexp.MustCompile(`,\s*([}\]])`)
	return re.ReplaceAllString(jsonStr, "$1")
}

// ParseJSON 解析JSON字符串为map[string]interface{}
// 参数：
//   - jsonStr: JSON字符串
//
// 返回：
//   - map[string]interface{}: 解析后的对象
//   - error: 解析错误
func ParseJSON(jsonStr string) (map[string]interface{}, error) {
	// 首先尝试直接解析
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err == nil {
		return result, nil
	}

	// 如果直接解析失败，尝试清理后再解析
	cleanedJSON := CleanJSON(jsonStr)
	err = json.Unmarshal([]byte(cleanedJSON), &result)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	return result, nil
}

// ExtractField 从JSON字符串中提取特定字段
// 参数：
//   - jsonStr: JSON字符串
//   - fieldName: 要提取的字段名
//
// 返回：
//   - string: 提取的字段值
//   - error: 提取错误
func ExtractField(jsonStr, fieldName string) (string, error) {
	// 首先尝试解析为map
	jsonMap, err := ParseJSON(jsonStr)
	if err == nil {
		if value, ok := jsonMap[fieldName]; ok {
			switch v := value.(type) {
			case string:
				return v, nil
			default:
				// 如果不是字符串，转换为JSON字符串
				valueBytes, err := json.Marshal(v)
				if err != nil {
					return "", fmt.Errorf("无法序列化字段值: %v", err)
				}
				return string(valueBytes), nil
			}
		}
	}

	// 如果解析失败或字段不存在，尝试直接提取
	fieldPattern := fmt.Sprintf(`"%s"\s*:\s*"([^"\\]*(\\.[^"\\]*)*)"`, regexp.QuoteMeta(fieldName))
	re := regexp.MustCompile(fieldPattern)
	matches := re.FindStringSubmatch(jsonStr)
	if len(matches) > 1 {
		// 处理转义字符
		value := matches[1]
		value = strings.ReplaceAll(value, "\\\"", "\"")
		value = strings.ReplaceAll(value, "\\n", "\n")
		value = strings.ReplaceAll(value, "\\r", "\r")
		value = strings.ReplaceAll(value, "\\t", "\t")
		value = strings.ReplaceAll(value, "\\\\", "\\")
		return value, nil
	}

	return "", fmt.Errorf("未找到字段: %s", fieldName)
}
