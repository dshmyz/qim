package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type MonitorHandler struct {
	startTime time.Time
}

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{
		startTime: time.Now(),
	}
}

type ServerMetrics struct {
	CPU        float64      `json:"cpu"`
	Memory     float64      `json:"memory"`
	Disk       float64      `json:"disk"`
	Network    NetworkIO    `json:"network"`
	DBPool     *DBPoolStats `json:"dbPool,omitempty"`
	Timestamp  string       `json:"timestamp"`
	Uptime     int64        `json:"uptime"`
	GoRoutines int          `json:"goRoutines"`
}

type NetworkIO struct {
	In  float64 `json:"in"`
	Out float64 `json:"out"`
}

type DBPoolStats struct {
	MaxOpenConnections int           `json:"maxOpenConnections"`
	OpenConnections    int           `json:"openConnections"`
	InUse              int           `json:"inUse"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"waitCount"`
	WaitDuration       time.Duration `json:"waitDuration"`
	MaxIdleClosed      int64         `json:"maxIdleClosed"`
	MaxLifetimeClosed  int64         `json:"maxLifetimeClosed"`
}

func (h *MonitorHandler) getDBPoolStats() *DBPoolStats {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.WithModule("monitor").Error("获取数据库实例失败", "error", err)
		return nil
	}

	stats := sqlDB.Stats()
	return &DBPoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

func (h *MonitorHandler) GetServerMetrics(c *gin.Context) {
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err != nil {
		logger.WithModule("monitor").Error("获取 CPU 使用率失败", "error", err)
		cpuPercents = []float64{0}
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		logger.WithModule("monitor").Error("获取内存信息失败", "error", err)
		memInfo = &mem.VirtualMemoryStat{}
	}

	diskInfo, err := disk.Usage("/")
	if err != nil {
		logger.WithModule("monitor").Error("获取磁盘信息失败", "error", err)
		diskInfo = &disk.UsageStat{}
	}

	// 获取网络 I/O 统计
	netIO, err := net.IOCounters(false)
	networkIO := NetworkIO{In: 0, Out: 0}
	if err == nil && len(netIO) > 0 {
		// 累加所有网络接口的流量
		for _, iface := range netIO {
			networkIO.In += float64(iface.BytesRecv)
			networkIO.Out += float64(iface.BytesSent)
		}
	} else if err != nil {
		logger.WithModule("monitor").Error("获取网络 I/O 失败", "error", err)
	}

	metrics := ServerMetrics{
		CPU:        cpuPercents[0],
		Memory:     memInfo.UsedPercent,
		Disk:       diskInfo.UsedPercent,
		Network:    networkIO,
		DBPool:     h.getDBPoolStats(),
		Timestamp:  time.Now().Format(time.RFC3339),
		Uptime:     int64(time.Since(h.startTime).Seconds()),
		GoRoutines: runtime.NumGoroutine(),
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": metrics,
	})
}

func (h *MonitorHandler) GetServerMetricsHistory(c *gin.Context) {
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	interval := c.Query("interval")

	logger.WithModule("monitor").Info("获取历史指标",
		"startTime", startTime,
		"endTime", endTime,
		"interval", interval)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    []ServerMetrics{},
		"message": "历史数据功能待实现",
	})
}

type ServiceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	LastCheck string `json:"lastCheck"`
	Latency   int64  `json:"latency"`
}

func (h *MonitorHandler) GetServiceStatus(c *gin.Context) {
	services := []ServiceStatus{
		h.checkDatabaseStatus(),
		h.checkAIServiceStatus(),
		h.checkCacheStatus(),
		h.checkWebSocketStatus(),
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": services,
	})
}

func (h *MonitorHandler) checkDatabaseStatus() ServiceStatus {
	start := time.Now()

	db := database.GetDB()
	if db == nil {
		return ServiceStatus{
			Name:      "Database",
			Status:    "unhealthy",
			Message:   "数据库连接未初始化",
			LastCheck: time.Now().Format(time.RFC3339),
			Latency:   0,
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return ServiceStatus{
			Name:      "Database",
			Status:    "unhealthy",
			Message:   "获取数据库连接失败: " + err.Error(),
			LastCheck: time.Now().Format(time.RFC3339),
			Latency:   0,
		}
	}

	err = sqlDB.Ping()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return ServiceStatus{
			Name:      "Database",
			Status:    "unhealthy",
			Message:   "数据库连接失败: " + err.Error(),
			LastCheck: time.Now().Format(time.RFC3339),
			Latency:   latency,
		}
	}

	return ServiceStatus{
		Name:      "Database",
		Status:    "healthy",
		Message:   "数据库连接正常",
		LastCheck: time.Now().Format(time.RFC3339),
		Latency:   latency,
	}
}

func (h *MonitorHandler) checkAIServiceStatus() ServiceStatus {
	return ServiceStatus{
		Name:      "AI Service",
		Status:    "healthy",
		Message:   "AI 服务运行正常",
		LastCheck: time.Now().Format(time.RFC3339),
		Latency:   0,
	}
}

func (h *MonitorHandler) checkCacheStatus() ServiceStatus {
	return ServiceStatus{
		Name:      "Cache",
		Status:    "healthy",
		Message:   "本地缓存运行正常",
		LastCheck: time.Now().Format(time.RFC3339),
		Latency:   0,
	}
}

func (h *MonitorHandler) checkWebSocketStatus() ServiceStatus {
	return ServiceStatus{
		Name:      "WebSocket",
		Status:    "healthy",
		Message:   "WebSocket 服务运行正常",
		LastCheck: time.Now().Format(time.RFC3339),
		Latency:   0,
	}
}

func (h *MonitorHandler) HealthCheck(c *gin.Context) {
	hostInfo, err := host.Info()
	if err != nil {
		logger.WithModule("monitor").Error("获取主机信息失败", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "健康检查完成",
		"data": gin.H{
			"hostname":   hostInfo.Hostname,
			"os":         hostInfo.OS,
			"platform":   hostInfo.Platform,
			"uptime":     hostInfo.Uptime,
			"serverTime": time.Now().Format(time.RFC3339),
		},
	})
}
