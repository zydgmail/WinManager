package controllers

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

var (
	// System metrics
	cpuUsageGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "agent_cpu_usage_percent",
		Help: "Current CPU usage percentage",
	})

	memoryUsageGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "agent_memory_usage_bytes",
		Help: "Current memory usage in bytes",
	})

	memoryTotalGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "agent_memory_total_bytes",
		Help: "Total memory in bytes",
	})

	goroutinesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "agent_goroutines_count",
		Help: "Number of goroutines",
	})

	// Agent metrics
	uptimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "agent_uptime_seconds",
		Help: "Agent uptime in seconds",
	})

	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "agent_requests_total",
		Help: "Total number of requests processed",
	}, []string{"method", "endpoint", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "agent_request_duration_seconds",
		Help: "Request duration in seconds",
	}, []string{"method", "endpoint"})
)

var startTime = time.Now()

// StartMetricsRecording starts the metrics collection goroutine
func StartMetricsRecording() {
	log.Info("Starting metrics recording")

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			recordSystemMetrics()
		}
	}()
}

// recordSystemMetrics collects and records system metrics
func recordSystemMetrics() {
	// Record CPU usage
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		cpuUsageGauge.Set(cpuPercent[0])
	} else {
		log.WithError(err).Debug("Failed to get CPU usage")
	}

	// Record memory usage
	if memInfo, err := mem.VirtualMemory(); err == nil {
		memoryUsageGauge.Set(float64(memInfo.Used))
		memoryTotalGauge.Set(float64(memInfo.Total))
	} else {
		log.WithError(err).Debug("Failed to get memory info")
	}

	// Record goroutines count
	goroutinesGauge.Set(float64(runtime.NumGoroutine()))

	// Record uptime
	uptimeGauge.Set(time.Since(startTime).Seconds())
}

// RecordRequest records metrics for HTTP requests
func RecordRequest(method, endpoint, status string, duration time.Duration) {
	requestsTotal.WithLabelValues(method, endpoint, status).Inc()
	requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// MetricsMiddleware returns a Gin middleware for recording request metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		status := fmt.Sprintf("%d", c.Writer.Status())
		
		RecordRequest(c.Request.Method, c.FullPath(), status, duration)
	}
}
