package domain

import "time"

// DashboardData contient les donnÃ©es pour le dashboard
type DashboardData struct {
	Title           string
	Status          string
	Time            string
	Groups          []MachineGroup
	TotalMachines   int
	RefreshInterval int
	Role            string
	CSRFToken       string
}

// MachineDetailData contient les donnÃ©es pour la page dÃ©tail d'une machine
type MachineDetailData struct {
	Machine   Machine
	Time      string
	Status    string
	Role      string
	CSRFToken string
}

// AuditLogData contient les donnÃ©es pour la page d'audit
type AuditLogData struct {
	Logs      []AuditLog
	Time      string
	Role      string
	CSRFToken string
}

// AuditLog reprÃ©sente une entrÃ©e de journal d'audit
type AuditLog struct {
	Timestamp time.Time `json:"timestamp"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	IPAddress string    `json:"ip_address"`
}
