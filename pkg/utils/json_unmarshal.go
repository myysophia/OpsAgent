package utils

import (
	"encoding/json"
	"fmt"
)

// JSONUnmarshal 是对标准 json.Unmarshal 的包装，提供更好的错误处理
// 参数：
//   - data: JSON字节数组
//   - v: 要解析到的目标对象
//
// 返回：
//   - error: 解析错误
func JSONUnmarshal(data []byte, v interface{}) error {
	// 首先尝试直接解析
	err := json.Unmarshal(data, v)
	if err == nil {
		return nil
	}
	
	// 如果直接解析失败，尝试清理后再解析
	cleanedJSON := CleanJSON(string(data))
	err = json.Unmarshal([]byte(cleanedJSON), v)
	if err != nil {
		return fmt.Errorf("解析JSON失败: %v", err)
	}
	
	return nil
}
