package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MQ 消费延迟（毫秒）
	MQConsumeLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mq_consume_latency_ms",
			Help:    "MQ message consumption latency in milliseconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 10), // 10ms to ~10s
		},
		[]string{"routing_key", "queue"},
	)

	// Agent 调用延迟（毫秒）
	AgentCallLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agent_call_latency_ms",
			Help:    "Agent service call latency in milliseconds",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100ms to ~100s
		},
		[]string{"endpoint", "status"},
	)

	// 数据库查询延迟（秒）
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
		},
		[]string{"operation", "table"},
	)

	// HTTP 请求延迟（秒）
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
		},
		[]string{"method", "path", "status"},
	)

	// 任务生成计数
	TaskGenerationCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_generation_count",
			Help: "Total number of tasks generated",
		},
		[]string{"source"}, // source: email, text, habit, project
	)

	// 邮件处理计数
	EmailProcessedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "email_processed_count",
			Help: "Total number of emails processed",
		},
		[]string{"status"}, // status: success, failed
	)

	// 慢查询计数
	SlowQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_slow_query_total",
			Help: "Total number of slow database queries (threshold: 100ms)",
		},
		[]string{"sql"}, // SQL 语句（截断到前 200 字符）
	)
)

// RecordMQConsumeLatency 记录 MQ 消费延迟
func RecordMQConsumeLatency(routingKey, queue string, duration time.Duration) {
	MQConsumeLatency.WithLabelValues(routingKey, queue).Observe(float64(duration.Milliseconds()))
}

// RecordAgentCallLatency 记录 Agent 调用延迟
func RecordAgentCallLatency(endpoint, status string, duration time.Duration) {
	AgentCallLatency.WithLabelValues(endpoint, status).Observe(float64(duration.Milliseconds()))
}

// RecordDBQueryDuration 记录数据库查询延迟
func RecordDBQueryDuration(operation, table string, duration time.Duration) {
	DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordHTTPRequestDuration 记录 HTTP 请求延迟
func RecordHTTPRequestDuration(method, path, status string, duration time.Duration) {
	HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
}

// IncrementTaskGeneration 增加任务生成计数
func IncrementTaskGeneration(source string) {
	TaskGenerationCount.WithLabelValues(source).Inc()
}

// IncrementEmailProcessed 增加邮件处理计数
func IncrementEmailProcessed(status string) {
	EmailProcessedCount.WithLabelValues(status).Inc()
}

// IncrementSlowQuery 增加慢查询计数
func IncrementSlowQuery(sql string, duration time.Duration) {
	// 截断 SQL 语句（避免标签值过长）
	if len(sql) > 200 {
		sql = sql[:200] + "..."
	}
	SlowQueryTotal.WithLabelValues(sql).Inc()
}

