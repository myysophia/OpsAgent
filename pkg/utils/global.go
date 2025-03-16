package utils

import (
	"sync"
)

var (
	// 全局变量映射
	globalVars = make(map[string]interface{})
	// 互斥锁，保证并发安全
	globalMutex sync.RWMutex
)

// SetGlobalVar 设置全局变量
func SetGlobalVar(key string, value interface{}) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalVars[key] = value
}

// GetGlobalVar 获取全局变量
func GetGlobalVar(key string) (interface{}, bool) {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	value, ok := globalVars[key]
	return value, ok
}

// RemoveGlobalVar 删除全局变量
func RemoveGlobalVar(key string) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	delete(globalVars, key)
}

// ClearGlobalVars 清除所有全局变量
func ClearGlobalVars() {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalVars = make(map[string]interface{})
} 