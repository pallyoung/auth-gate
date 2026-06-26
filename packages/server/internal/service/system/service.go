package system

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/pallyoung/auth-gate/packages/server/internal/router"
)

// Stats holds a point-in-time snapshot of system and application metrics.
type Stats struct {
	// System
	Uptime   int64  `json:"uptime_seconds"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Platform string `json:"platform"`
	Kernel   string `json:"kernel_version"`

	// CPU
	CPUCores int     `json:"cpu_cores"`
	CPUUsage float64 `json:"cpu_usage_percent"`

	// Memory
	MemTotal uint64  `json:"mem_total_bytes"`
	MemUsed  uint64  `json:"mem_used_bytes"`
	MemUsage float64 `json:"mem_usage_percent"`

	// Disk
	DiskTotal uint64  `json:"disk_total_bytes"`
	DiskUsed  uint64  `json:"disk_used_bytes"`
	DiskUsage float64 `json:"disk_usage_percent"`

	// Go runtime
	Goroutines   int     `json:"goroutines"`
	HeapAlloc    uint64  `json:"heap_alloc_bytes"`
	HeapInuse    uint64  `json:"heap_inuse_bytes"`
	GCCount      uint32  `json:"gc_count"`
	GCPauseTotal float64 `json:"gc_pause_total_ms"`

	// Application
	ActiveRoutes int `json:"active_routes"`
	TotalRoutes  int `json:"total_routes"`

	// Go info
	GoVersion string `json:"go_version"`
}

// Service collects system and application metrics.
type Service struct {
	startTime time.Time
	routerMgr *router.Manager
}

// NewService creates a new system metrics service.
func NewService(startTime time.Time, routerMgr *router.Manager) *Service {
	return &Service{startTime: startTime, routerMgr: routerMgr}
}

// Stats returns a point-in-time snapshot of all metrics.
func (s *Service) Stats() (*Stats, error) {
	st := &Stats{
		Uptime:    int64(time.Since(s.startTime).Seconds()),
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
	}

	// Host info
	if info, err := host.Info(); err == nil {
		st.Hostname = info.Hostname
		st.OS = info.OS
		st.Platform = info.Platform
		st.Kernel = info.KernelVersion
		st.Uptime = int64(info.Uptime)
	}

	// CPU
	st.CPUCores = runtime.NumCPU()
	if percents, err := cpu.Percent(500*time.Millisecond, false); err == nil && len(percents) > 0 {
		st.CPUUsage = percents[0]
	}

	// Memory
	if vmem, err := mem.VirtualMemory(); err == nil {
		st.MemTotal = vmem.Total
		st.MemUsed = vmem.Used
		st.MemUsage = vmem.UsedPercent
	}

	// Disk
	if usage, err := disk.Usage("/"); err == nil {
		st.DiskTotal = usage.Total
		st.DiskUsed = usage.Used
		st.DiskUsage = usage.UsedPercent
	}

	// Go runtime
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	st.Goroutines = runtime.NumGoroutine()
	st.HeapAlloc = m.HeapAlloc
	st.HeapInuse = m.HeapInuse
	st.GCCount = m.NumGC
	st.GCPauseTotal = float64(m.PauseTotalNs) / 1e6 // ns → ms

	// Application routes
	if s.routerMgr != nil {
		routes := s.routerMgr.GetRoutes()
		st.TotalRoutes = len(routes)
		for _, r := range routes {
			if r.Enabled {
				st.ActiveRoutes++
			}
		}
	}

	return st, nil
}
