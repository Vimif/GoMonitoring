package domain

import "time"

// Machine reprÃ©sente une machine surveillÃ©e
type Machine struct {
	ID        string
	Name      string
	Host      string
	Port      int
	User      string
	KeyPath   string
	Group     string
	OSType    string    // "linux", "windows", "unknown"
	Status    string    // "online", "offline", "error"
	LastCheck time.Time `json:"last_check"`
	System    SystemInfo      `json:"system"`
	CPU       CPUInfo         `json:"cpu"`
	Memory    MemoryInfo      `json:"memory"`
	Disks     []DiskInfo      `json:"disks"`
	Network   NetworkStats    `json:"network"`
	DiskIO    DiskStats       `json:"disk_io"`
	Services  []ServiceStatus `json:"services"`
}

// ServiceStatus reprÃ©sente l'Ã©tat d'un service systÃ¨me
type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "active", "inactive", "failed", "unknown"
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

// NetworkStats contient les statistiques rÃ©seau
type NetworkStats struct {
	RxBytes uint64  `json:"rx_bytes"`
	TxBytes uint64  `json:"tx_bytes"`
	RxRate  float64 `json:"rx_rate"` // Octets/s
	TxRate  float64 `json:"tx_rate"` // Octets/s
}

// DiskStats contient les statistiques d'E/S disque
type DiskStats struct {
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	ReadRate   float64 `json:"read_rate"`  // Octets/s
	WriteRate  float64 `json:"write_rate"` // Octets/s
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
