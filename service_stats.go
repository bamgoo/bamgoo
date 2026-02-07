package bamgoo

// ServiceStats contains service statistics.
type ServiceStats struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	NumRequests  int    `json:"num_requests"`
	NumErrors    int    `json:"num_errors"`
	TotalLatency int64  `json:"total_latency_ms"`
	AvgLatency   int64  `json:"avg_latency_ms"`
}
