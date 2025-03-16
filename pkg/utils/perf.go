package utils

import (
	"fmt"
	"go.uber.org/zap"
	"sort"
	"sync"
	"time"
)

// PerfStats 性能统计结构体
// 用于收集和分析系统各个部分的性能数据
type PerfStats struct {
	mu            sync.RWMutex
	metrics       map[string][]time.Duration // 存储每个操作的耗时记录
	startTimes    map[string]time.Time       // 存储操作的开始时间
	logger        *zap.Logger                // 日志记录器
	enableLogging bool                       // 是否启用日志记录
	timers        map[string]time.Duration
	callCounts    map[string]int64
	lastResetTime time.Time
}

// 全局性能统计实例
var (
	globalPerfStats *PerfStats
	once            sync.Once
)

// GetPerfStats 获取全局性能统计实例
// 返回：
//   - *PerfStats: 全局性能统计实例
func GetPerfStats() *PerfStats {
	once.Do(func() {
		globalPerfStats = &PerfStats{
			metrics:       make(map[string][]time.Duration),
			startTimes:    make(map[string]time.Time),
			enableLogging: true,
			timers:        make(map[string]time.Duration),
			callCounts:    make(map[string]int64),
			lastResetTime: time.Now(),
		}
	})
	return globalPerfStats
}

// SetLogger 设置日志记录器
// 参数：
//   - logger: zap日志记录器
func (p *PerfStats) SetLogger(logger *zap.Logger) {
	p.logger = logger
}

// SetEnableLogging 设置是否启用日志记录
// 参数：
//   - enable: 是否启用
func (p *PerfStats) SetEnableLogging(enable bool) {
	p.enableLogging = enable
}

// StartTimer 开始计时特定操作
// 参数：
//   - operation: 操作名称
func (p *PerfStats) StartTimer(operation string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTimes[operation] = time.Now()
	p.timers[operation] = 0
	
	if p.enableLogging && p.logger != nil {
		p.logger.Debug("开始计时操作",
			zap.String("operation", operation),
			zap.Time("start_time", p.startTimes[operation]),
		)
	}
}

// StopTimer 停止计时特定操作并记录耗时
// 参数：
//   - operation: 操作名称
// 返回：
//   - time.Duration: 操作耗时
func (p *PerfStats) StopTimer(operation string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	startTime, exists := p.startTimes[operation]
	if !exists {
		if p.enableLogging && p.logger != nil {
			p.logger.Warn("尝试停止未开始的计时操作",
				zap.String("operation", operation),
			)
		}
		return 0
	}
	
	elapsed := time.Since(startTime)
	delete(p.startTimes, operation)
	
	if _, exists := p.metrics[operation]; !exists {
		p.metrics[operation] = []time.Duration{}
	}
	p.metrics[operation] = append(p.metrics[operation], elapsed)
	
	if _, exists := p.timers[operation]; !exists {
		p.timers[operation] = 0
	}
	p.timers[operation] = elapsed
	
	if p.enableLogging && p.logger != nil {
		p.logger.Debug("完成计时操作",
			zap.String("operation", operation),
			zap.Duration("elapsed", elapsed),
		)
	}
	
	return elapsed
}

// RecordMetric 直接记录一个性能指标
// 参数：
//   - operation: 操作名称
//   - duration: 操作耗时
func (p *PerfStats) RecordMetric(operation string, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if _, exists := p.metrics[operation]; !exists {
		p.metrics[operation] = []time.Duration{}
	}
	p.metrics[operation] = append(p.metrics[operation], duration)
	
	if p.enableLogging && p.logger != nil {
		p.logger.Debug("记录性能指标",
			zap.String("operation", operation),
			zap.Duration("duration", duration),
		)
	}
}

// GetMetrics 获取所有性能指标
// 返回：
//   - map[string][]time.Duration: 所有操作的耗时记录
func (p *PerfStats) GetMetrics() map[string][]time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// 创建副本以避免并发问题
	metrics := make(map[string][]time.Duration)
	for op, durations := range p.metrics {
		metrics[op] = append([]time.Duration{}, durations...)
	}
	
	return metrics
}

// GetMetricStats 获取特定操作的统计信息
// 参数：
//   - operation: 操作名称
// 返回：
//   - min: 最小耗时
//   - max: 最大耗时
//   - avg: 平均耗时
//   - p95: 95百分位耗时
//   - p99: 99百分位耗时
//   - count: 操作次数
//   - total: 总耗时
func (p *PerfStats) GetMetricStats(operation string) (min, max, avg, p95, p99 time.Duration, count int, total time.Duration) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	durations, exists := p.metrics[operation]
	if !exists || len(durations) == 0 {
		return 0, 0, 0, 0, 0, 0, 0
	}
	
	count = len(durations)
	
	// 创建副本并排序
	sortedDurations := make([]time.Duration, count)
	copy(sortedDurations, durations)
	sort.Slice(sortedDurations, func(i, j int) bool {
		return sortedDurations[i] < sortedDurations[j]
	})
	
	min = sortedDurations[0]
	max = sortedDurations[count-1]
	
	// 计算总和和平均值
	for _, d := range durations {
		total += d
	}
	avg = total / time.Duration(count)
	
	// 计算百分位数
	p95Index := int(float64(count) * 0.95)
	p99Index := int(float64(count) * 0.99)
	
	if p95Index >= count {
		p95Index = count - 1
	}
	if p99Index >= count {
		p99Index = count - 1
	}
	
	p95 = sortedDurations[p95Index]
	p99 = sortedDurations[p99Index]
	
	return
}

// ResetMetrics 重置所有性能指标
func (p *PerfStats) ResetMetrics() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.metrics = make(map[string][]time.Duration)
	p.startTimes = make(map[string]time.Time)
	
	if p.enableLogging && p.logger != nil {
		p.logger.Info("重置所有性能指标")
	}
}

// PrintStats 打印所有性能统计信息
// 返回：
//   - string: 格式化的统计信息
func (p *PerfStats) PrintStats() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if len(p.metrics) == 0 {
		return "没有收集到性能指标"
	}
	
	var result string
	result = "性能统计信息:\n"
	result += "------------------------------------------------------------\n"
	result += fmt.Sprintf("%-30s %-10s %-10s %-10s %-10s %-10s %-10s\n", 
		"操作", "次数", "平均", "最小", "最大", "P95", "P99")
	result += "------------------------------------------------------------\n"
	
	// 按操作名称排序
	operations := make([]string, 0, len(p.metrics))
	for op := range p.metrics {
		operations = append(operations, op)
	}
	sort.Strings(operations)
	
	for _, op := range operations {
		min, max, avg, p95, p99, count, _ := p.GetMetricStats(op)
		result += fmt.Sprintf("%-30s %-10d %-10s %-10s %-10s %-10s %-10s\n",
			op, count, 
			formatDuration(avg), 
			formatDuration(min), 
			formatDuration(max), 
			formatDuration(p95), 
			formatDuration(p99))
	}
	result += "------------------------------------------------------------\n"
	
	return result
}

// formatDuration 格式化时间间隔为易读形式
// 参数：
//   - d: 时间间隔
// 返回：
//   - string: 格式化后的字符串
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.2fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// TraceFunc 是一个辅助函数，用于跟踪函数执行时间
// 使用方法：defer utils.GetPerfStats().TraceFunc("函数名称")()
// 参数：
//   - operation: 操作名称
// 返回：
//   - func(): 在函数结束时调用的函数
func (p *PerfStats) TraceFunc(operation string) func() {
	p.StartTimer(operation)
	return func() {
		p.StopTimer(operation)
	}
}

// GetStats 获取所有性能统计信息
func (p *PerfStats) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	
	// 添加计时器信息
	timers := make(map[string]time.Duration)
	for name, duration := range p.timers {
		timers[name] = duration
	}
	stats["timers"] = timers

	// 添加调用次数信息
	callCounts := make(map[string]int64)
	for name, count := range p.callCounts {
		callCounts[name] = count
	}
	stats["callCounts"] = callCounts

	// 添加最后重置时间
	stats["lastResetTime"] = p.lastResetTime

	return stats
}

// Reset 重置所有性能统计信息
func (p *PerfStats) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 清空计时器
	p.timers = make(map[string]time.Duration)
	
	// 清空调用次数
	p.callCounts = make(map[string]int64)
	
	// 更新最后重置时间
	p.lastResetTime = time.Now()
} 