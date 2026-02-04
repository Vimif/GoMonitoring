package models

import "time"

// Machine reprÃ©sente une machine surveillÃ©e
type Machine struct {
	ID        string
	Name      string
	Host      string
	Port      int
	User      string
	KeyPath   string          // Champ ajoutÃ© pour l'Ã©dition
	Group     string          // Nouveau champ
	OSType    string          // "linux", "windows", "unknown"
	Status    string          // "online", "offline", "error"
	LastCheck time.Time       `json:"last_check"`
	System    SystemInfo      `json:"system"`
	CPU       CPUInfo         `json:"cpu"`
	Memory    MemoryInfo      `json:"memory"`
	Disks     []DiskInfo      `json:"disks"`
	Network   NetworkStats    `json:"network"`
	DiskIO    DiskStats       `json:"disk_io"`
	Services  []ServiceStatus `json:"services"` // Liste des services
}

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "active", "inactive", "failed", "unknown"
}

type NetworkStats struct {
	RxBytes uint64  `json:"rx_bytes"`
	TxBytes uint64  `json:"tx_bytes"`
	RxRate  float64 `json:"rx_rate"` // Octets/s
	TxRate  float64 `json:"tx_rate"` // Octets/s
}

type DiskStats struct {
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	ReadRate   float64 `json:"read_rate"`  // Octets/s
	WriteRate  float64 `json:"write_rate"` // Octets/s
}

// SystemInfo contient les informations gÃ©nÃ©rales du systÃ¨me
type SystemInfo struct {
	Hostname     string
	OS           string
	Kernel       string
	Architecture string
	Uptime       string
	BootTime     time.Time
}

// CPUInfo contient les informations du processeur
type CPUInfo struct {
	Model        string
	Cores        int
	Threads      int
	MHz          float64
	UsagePercent float64
}

// MemoryInfo contient les informations de la mÃ©moire
type MemoryInfo struct {
	Total       uint64
	Used        uint64
	Free        uint64
	Available   uint64
	UsedPercent float64
}

// DiskInfo contient les informations d'un disque
type DiskInfo struct {
	Device      string
	MountPoint  string
	FSType      string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	DriveType   string // SSD, HDD, Unknown
}

// Partition reprÃ©sente une partition avec ses options
type Partition struct {
	Name       string
	Size       uint64
	Type       string
	MountPoint string
	Options    []string
}

// DirectoryListing reprÃ©sente le contenu d'un rÃ©pertoire
type DirectoryListing struct {
	Path    string
	Parent  string
	Entries []DirectoryEntry
}

// DirectoryEntry reprÃ©sente un fichier ou dossier
type DirectoryEntry struct {
	Name        string
	Path        string
	IsDir       bool
	Size        int64
	ModTime     time.Time
	Permissions string
	Owner       string
	Group       string
}

// MachineGroup reprÃ©sente un groupe de machines
type MachineGroup struct {
	Name     string
	Machines []Machine
}

// DashboardData contient les donnÃ©es pour le dashboard
type DashboardData struct {
	Title           string
	Status          string
	Time            string
	Groups          []MachineGroup
	TotalMachines   int
	RefreshInterval int
	Role            string
	CSRFToken       string // Token CSRF pour les formulaires
}

// MetricPoint reprÃ©sente un point de mesure dans l'historique
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

// MachineDetailData contient les donnÃ©es pour la page dÃ©tail
type MachineDetailData struct {
	Machine   Machine
	Time      string
	Status    string
	Role      string
	CSRFToken string // Token CSRF pour les formulaires
}
