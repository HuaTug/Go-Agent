package main

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// MonitoringSystem 监控系统
type MonitoringSystem struct {
	Ctx             context.Context
	MetricsRegistry *prometheus.Registry
	TracerProvider  trace.TracerProvider
	Tracer          trace.Tracer
	Metrics         *SystemMetrics
	AlertManager    *AlertManager
	LogCollector    *LogCollector
	mu              sync.RWMutex
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	// Prometheus指标
	RequestsTotal    prometheus.Counter
	RequestsDuration prometheus.Histogram
	LLMCallsTotal    prometheus.Counter
	ToolCallsTotal   prometheus.Counter
	ErrorsTotal      prometheus.Counter
	CacheHitsTotal   prometheus.Counter
	CacheMissesTotal prometheus.Counter
	MemoryUsage      prometheus.Gauge
	GoroutinesCount  prometheus.Gauge
	QueueSize        prometheus.Gauge
	ActiveWorkers    prometheus.Gauge

	// 自定义业务指标
	TaskCompletionTime  prometheus.Histogram
	ToolExecutionTime   prometheus.Histogram
	TokenUsage          prometheus.Counter
	RAGSearchLatency    prometheus.Histogram
	VectorSearchLatency prometheus.Histogram
}

// AlertManager 告警管理器
type AlertManager struct {
	Alerts        map[string]*AlertRule
	AlertChannels []AlertChannel
	mu            sync.RWMutex
}

// AlertRule 告警规则
type AlertRule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Condition string        `json:"condition"`
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
	Severity  string        `json:"severity"` // critical, warning, info
	Enabled   bool          `json:"enabled"`
	CreatedAt time.Time     `json:"created_at"`
}

// AlertChannel 告警通道
type AlertChannel struct {
	Type    string                 `json:"type"` // webhook, email, slack, etc.
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

// LogCollector 日志收集器
type LogCollector struct {
	LogBuffer []*LogEntry
	MaxSize   int
	mu        sync.RWMutex
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`     // debug, info, warn, error
	Component string                 `json:"component"` // agent, llm, tool, etc.
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
}

// TraceContext 追踪上下文
type TraceContext struct {
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	Attributes map[string]interface{} `json:"attributes"`
}

// NewMonitoringSystem 创建监控系统
func NewMonitoringSystem(ctx context.Context, serviceName string) (*MonitoringSystem, error) {
	// 创建指标注册表
	registry := prometheus.NewRegistry()

	// 初始化指标
	metrics := &SystemMetrics{
		RequestsTotal: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_requests_total",
			Help: "Total number of agent requests",
		}),
		RequestsDuration: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_request_duration_seconds",
			Help:    "Duration of agent requests in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		LLMCallsTotal: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_llm_calls_total",
			Help: "Total number of LLM calls",
		}),
		ToolCallsTotal: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_tool_calls_total",
			Help: "Total number of tool calls",
		}),
		ErrorsTotal: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_errors_total",
			Help: "Total number of errors",
		}),
		CacheHitsTotal: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_cache_hits_total",
			Help: "Total number of cache hits",
		}),
		CacheMissesTotal: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_cache_misses_total",
			Help: "Total number of cache misses",
		}),
		MemoryUsage: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		}),
		GoroutinesCount: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_goroutines_count",
			Help: "Current number of goroutines",
		}),
		QueueSize: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_queue_size",
			Help: "Current task queue size",
		}),
		ActiveWorkers: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_active_workers",
			Help: "Current number of active workers",
		}),
		TaskCompletionTime: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_task_completion_time_seconds",
			Help:    "Time taken to complete tasks",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
		}),
		ToolExecutionTime: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_tool_execution_time_seconds",
			Help:    "Time taken to execute tools",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
		}),
		TokenUsage: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_token_usage_total",
			Help: "Total tokens used by LLM",
		}),
		RAGSearchLatency: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_rag_search_latency_seconds",
			Help:    "RAG search latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		}),
		VectorSearchLatency: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_vector_search_latency_seconds",
			Help:    "Vector search latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		}),
	}

	// 初始化追踪器
	tracerProvider, err := initTracer(serviceName)
	if err != nil {
		return nil, err
	}

	tracer := tracerProvider.Tracer("agent-system")

	system := &MonitoringSystem{
		Ctx:             ctx,
		MetricsRegistry: registry,
		TracerProvider:  tracerProvider,
		Tracer:          tracer,
		Metrics:         metrics,
		AlertManager: &AlertManager{
			Alerts:        make(map[string]*AlertRule),
			AlertChannels: make([]AlertChannel, 0),
		},
		LogCollector: &LogCollector{
			LogBuffer: make([]*LogEntry, 0),
			MaxSize:   10000,
		},
	}

	// 启动后台任务
	go system.collectSystemMetrics()
	go system.checkAlerts()

	return system, nil
}

// initTracer 初始化追踪器
func initTracer(serviceName string) (trace.TracerProvider, error) {
	// 创建Jaeger导出器
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		return nil, err
	}

	// 创建追踪器提供者
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("environment", "production"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// StartSpan 开始追踪span
func (ms *MonitoringSystem) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ms.Tracer.Start(ctx, name, opts...)
}

// RecordRequest 记录请求指标
func (ms *MonitoringSystem) RecordRequest(method string, duration time.Duration, success bool) {
	ms.Metrics.RequestsTotal.Inc()
	ms.Metrics.RequestsDuration.Observe(duration.Seconds())

	if !success {
		ms.Metrics.ErrorsTotal.Inc()
	}
}

// RecordLLMCall 记录LLM调用指标
func (ms *MonitoringSystem) RecordLLMCall(duration time.Duration, tokens int, success bool) {
	ms.Metrics.LLMCallsTotal.Inc()
	ms.Metrics.TokenUsage.Add(float64(tokens))

	if !success {
		ms.Metrics.ErrorsTotal.Inc()
	}
}

// RecordToolCall 记录工具调用指标
func (ms *MonitoringSystem) RecordToolCall(toolName string, duration time.Duration, success bool) {
	ms.Metrics.ToolCallsTotal.Inc()
	ms.Metrics.ToolExecutionTime.Observe(duration.Seconds())

	if !success {
		ms.Metrics.ErrorsTotal.Inc()
	}
}

// RecordCacheHit 记录缓存命中
func (ms *MonitoringSystem) RecordCacheHit() {
	ms.Metrics.CacheHitsTotal.Inc()
}

// RecordCacheMiss 记录缓存未命中
func (ms *MonitoringSystem) RecordCacheMiss() {
	ms.Metrics.CacheMissesTotal.Inc()
}

// RecordRAGSearch 记录RAG搜索指标
func (ms *MonitoringSystem) RecordRAGSearch(duration time.Duration, results int) {
	ms.Metrics.RAGSearchLatency.Observe(duration.Seconds())
}

// RecordVectorSearch 记录向量搜索指标
func (ms *MonitoringSystem) RecordVectorSearch(duration time.Duration) {
	ms.Metrics.VectorSearchLatency.Observe(duration.Seconds())
}

// Log 记录日志
func (ms *MonitoringSystem) Log(level, component, message string, fields map[string]interface{}, traceCtx *TraceContext) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
		Fields:    fields,
	}

	if traceCtx != nil {
		entry.TraceID = traceCtx.TraceID
		entry.SpanID = traceCtx.SpanID
	}

	ms.LogCollector.mu.Lock()
	defer ms.LogCollector.mu.Unlock()

	// 检查缓冲区大小
	if len(ms.LogCollector.LogBuffer) >= ms.LogCollector.MaxSize {
		// 移除最旧的日志条目
		ms.LogCollector.LogBuffer = ms.LogCollector.LogBuffer[1:]
	}

	ms.LogCollector.LogBuffer = append(ms.LogCollector.LogBuffer, entry)
}

// AddAlertRule 添加告警规则
func (ms *MonitoringSystem) AddAlertRule(rule *AlertRule) error {
	ms.AlertManager.mu.Lock()
	defer ms.AlertManager.mu.Unlock()

	if _, exists := ms.AlertManager.Alerts[rule.ID]; exists {
		return fmt.Errorf("alert rule with ID %s already exists", rule.ID)
	}

	ms.AlertManager.Alerts[rule.ID] = rule
	return nil
}

// AddAlertChannel 添加告警通道
func (ms *MonitoringSystem) AddAlertChannel(channel AlertChannel) {
	ms.AlertManager.mu.Lock()
	defer ms.AlertManager.mu.Unlock()

	ms.AlertManager.AlertChannels = append(ms.AlertManager.AlertChannels, channel)
}

// collectSystemMetrics 收集系统指标
func (ms *MonitoringSystem) collectSystemMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 收集内存使用情况
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			ms.Metrics.MemoryUsage.Set(float64(m.Alloc))

			// 收集goroutine数量
			ms.Metrics.GoroutinesCount.Set(float64(runtime.NumGoroutine()))

		case <-ms.Ctx.Done():
			return
		}
	}
}

// checkAlerts 检查告警
func (ms *MonitoringSystem) checkAlerts() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.evaluateAlertRules()
		case <-ms.Ctx.Done():
			return
		}
	}
}

// evaluateAlertRules 评估告警规则
func (ms *MonitoringSystem) evaluateAlertRules() {
	ms.AlertManager.mu.RLock()
	defer ms.AlertManager.mu.RUnlock()

	for _, rule := range ms.AlertManager.Alerts {
		if !rule.Enabled {
			continue
		}

		// 这里应该实现具体的告警条件评估逻辑
		// 简化实现：记录日志
		ms.Log("info", "alert-manager", fmt.Sprintf("Evaluating alert rule: %s", rule.Name), nil, nil)
	}
}

// GetMetrics 获取当前指标值
func (ms *MonitoringSystem) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// 收集Prometheus指标
	metrics["requests_total"] = getCounterValue(ms.Metrics.RequestsTotal)
	metrics["llm_calls_total"] = getCounterValue(ms.Metrics.LLMCallsTotal)
	metrics["tool_calls_total"] = getCounterValue(ms.Metrics.ToolCallsTotal)
	metrics["errors_total"] = getCounterValue(ms.Metrics.ErrorsTotal)
	metrics["cache_hits_total"] = getCounterValue(ms.Metrics.CacheHitsTotal)
	metrics["cache_misses_total"] = getCounterValue(ms.Metrics.CacheMissesTotal)
	metrics["memory_usage_bytes"] = getGaugeValue(ms.Metrics.MemoryUsage)
	metrics["goroutines_count"] = getGaugeValue(ms.Metrics.GoroutinesCount)

	return metrics
}

// GetLogs 获取日志
func (ms *MonitoringSystem) GetLogs(level, component string, limit int) []*LogEntry {
	ms.LogCollector.mu.RLock()
	defer ms.LogCollector.mu.RUnlock()

	var filteredLogs []*LogEntry
	count := 0

	// 反向遍历（最新的在前）
	for i := len(ms.LogCollector.LogBuffer) - 1; i >= 0 && count < limit; i-- {
		entry := ms.LogCollector.LogBuffer[i]

		if (level == "" || entry.Level == level) &&
			(component == "" || entry.Component == component) {
			filteredLogs = append(filteredLogs, entry)
			count++
		}
	}

	return filteredLogs
}

// StartMetricsServer 启动指标服务器
func (ms *MonitoringSystem) StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.HandlerFor(ms.MetricsRegistry, promhttp.HandlerOpts{}))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return http.ListenAndServe(addr, nil)
}

// getCounterValue 获取计数器值
func getCounterValue(counter prometheus.Counter) float64 {
	dto := &io_prometheus_client.Metric{}
	if err := counter.Write(dto); err != nil {
		return 0
	}
	if dto.Counter != nil {
		return dto.Counter.GetValue()
	}
	return 0
}

// getGaugeValue 获取仪表值
func getGaugeValue(gauge prometheus.Gauge) float64 {
	dto := &io_prometheus_client.Metric{}
	if err := gauge.Write(dto); err != nil {
		return 0
	}
	if dto.Gauge != nil {
		return dto.Gauge.GetValue()
	}
	return 0
}

// Shutdown 关闭监控系统
func (ms *MonitoringSystem) Shutdown() {
	if provider, ok := ms.TracerProvider.(*sdktrace.TracerProvider); ok {
		provider.Shutdown(ms.Ctx)
	}
}
