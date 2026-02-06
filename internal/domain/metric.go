package domain

import "time"

// MetricPoint représente un point de mesure dans l'historique
type MetricPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	CPU         float64   `json:"cpu"`
	MemoryUsed  uint64    `json:"memory_used"`
	MemoryTotal uint64    `json:"memory_total"`
	Status      string    `json:"status"`
	NetRxRate   float64   `json:"net_rx"`
	NetTxRate   float64   `json:"net_tx"`
	DiskRead    float64   `json:"disk_read"`
	DiskWrite   float64   `json:"disk_write"`
}
