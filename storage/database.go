package storage

import (
	"database/sql"
	"log"
	"time"

	"go-monitoring/models"

	_ "github.com/mattn/go-sqlite3"
)

// DB encapsule la connexion base de donnÃ©es
type DB struct {
	*sql.DB
}

// AuditLog reprÃ©sente une entrÃ©e dans le journal d'audit
type AuditLog struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Target    string    `json:"target"` // Cible de l'action (ex: machine-1, system)
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
}

// UserDB structure pour la base de donnÃ©es
type UserDB struct {
	Username       string    `json:"username"`
	PasswordHash   string    `json:"-"`
	Role           string    `json:"role"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	FailedAttempts int       `json:"failed_attempts"`
	LockedUntil    time.Time `json:"locked_until"`
}

// InitDB initialise la connexion SQLite et crÃ©e les tables
func InitDB(filepath string) (*DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS metrics (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        machine_id TEXT NOT NULL,
        timestamp DATETIME NOT NULL,
        cpu_usage REAL,
        memory_used INTEGER,
        memory_total INTEGER,
        status TEXT,
        net_rx_rate REAL DEFAULT 0,
        net_tx_rate REAL DEFAULT 0,
        disk_read_rate REAL DEFAULT 0,
        disk_write_rate REAL DEFAULT 0
    );
    CREATE INDEX IF NOT EXISTS idx_metrics_machine_time ON metrics(machine_id, timestamp);

    CREATE TABLE IF NOT EXISTS audit_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL,
        user TEXT NOT NULL,
        action TEXT NOT NULL,
        target TEXT,
        details TEXT,
        ip_address TEXT
    );
    CREATE TABLE IF NOT EXISTS audit_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL,
        user TEXT NOT NULL,
        action TEXT NOT NULL,
        target TEXT,
        details TEXT,
        ip_address TEXT
    );
    CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_logs(timestamp);

    CREATE TABLE IF NOT EXISTS users (
        username TEXT PRIMARY KEY,
        password_hash TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'user',
        is_active BOOLEAN NOT NULL DEFAULT 1,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        failed_attempts INTEGER DEFAULT 0,
        locked_until DATETIME
    );
    `

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	// Migrations (ignorer les erreurs si les colonnes existent dÃ©jÃ )
	// These ALTER TABLE statements are for backward compatibility if the table already exists without these columns.
	// In a real application, a more robust migration system would be used.
	db.Exec("ALTER TABLE metrics ADD COLUMN net_rx_rate REAL DEFAULT 0")
	db.Exec("ALTER TABLE metrics ADD COLUMN net_tx_rate REAL DEFAULT 0")
	db.Exec("ALTER TABLE metrics ADD COLUMN disk_read_rate REAL DEFAULT 0")
	db.Exec("ALTER TABLE metrics ADD COLUMN disk_write_rate REAL DEFAULT 0")

	return &DB{db}, nil
}

// SaveMetric sauvegarde une mesure
func (db *DB) SaveMetric(m models.Machine) error {
	query := `INSERT INTO metrics (
		machine_id, timestamp, cpu_usage, memory_used, memory_total, status,
		net_rx_rate, net_tx_rate, disk_read_rate, disk_write_rate
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		m.ID, m.LastCheck, m.CPU.UsagePercent, m.Memory.Used, m.Memory.Total, m.Status,
		m.Network.RxRate, m.Network.TxRate, m.DiskIO.ReadRate, m.DiskIO.WriteRate,
	)
	if err != nil {
		log.Printf("Erreur sauvegarde mÃ©trique %s: %v", m.ID, err)
	}
	return err
}

// GetHistory rÃ©cupÃ¨re l'historique d'une machine
func (db *DB) GetHistory(machineID string, duration time.Duration) ([]models.MetricPoint, error) {
	startTime := time.Now().Add(-duration)
	query := `SELECT timestamp, cpu_usage, memory_used, memory_total, status, net_rx_rate, net_tx_rate, disk_read_rate, disk_write_rate
			  FROM metrics WHERE machine_id = ? AND timestamp > ? ORDER BY timestamp ASC`

	rows, err := db.Query(query, machineID, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.MetricPoint
	for rows.Next() {
		var p models.MetricPoint
		var timestamp time.Time // Temporary variable for scanning timestamp
		if err := rows.Scan(&timestamp, &p.CPU, &p.MemoryUsed, &p.MemoryTotal, &p.Status, &p.NetRxRate, &p.NetTxRate, &p.DiskRead, &p.DiskWrite); err != nil {
			// Log the error and continue to the next row, or return the error if critical
			log.Printf("Error scanning row for machine %s: %v", machineID, err)
			continue
		}
		p.Timestamp = timestamp
		points = append(points, p)
	}
	return points, nil
}

// CleanupOldMetrics supprime les mÃ©triques plus vieilles que duration
func (db *DB) CleanupOldMetrics(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	_, err := db.Exec("DELETE FROM metrics WHERE timestamp < ?", cutoff)
	return err
}
