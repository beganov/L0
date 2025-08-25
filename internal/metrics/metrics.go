package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	KafkaMessagesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_messages_total",
			Help: "Общее количество обработанных сообщений из Kafka",
		})

	KafkaErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_errors_total",
			Help: "Количество ошибок при чтении из Kafka",
		})

	KafkaProcessDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kafka_process_duration_seconds",
			Help:    "Время обработки одного Kafka-сообщения",
			Buckets: prometheus.DefBuckets,
		})
)

var (
	DBErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Ошибки запросов в базу данных",
		})
)

var (
	CacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Количество попаданий в кэш",
		})

	CacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Количество промахов в кэше",
		})
)

var (
	HttpRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Количество HTTP-запросов",
		})

	HttpErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Ошибки HTTP",
		})

	HttpDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Время ответа API",
			Buckets: prometheus.DefBuckets,
		})
)

func Init() {
	prometheus.MustRegister(
		KafkaMessagesTotal, KafkaErrorsTotal, KafkaProcessDuration,
		DBErrorsTotal,
		CacheHits, CacheMisses,
		HttpRequestsTotal, HttpErrorsTotal, HttpDuration,
	)
}
