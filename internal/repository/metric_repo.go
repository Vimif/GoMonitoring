package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go-monitoring/internal/domain"
)

// MetricRepository implémente l'interface MetricRepository avec SQLite
type MetricRepository struct {
	db *sql.DB
}

// NewMetricRepository crée un nouveau repository pour les métriques
func NewMetricRepository(db *sql.DB) *MetricRepository {
	return &MetricRepository{
		db: db,
	}
}

// Save sauvegarde un point de métrique
func (r *MetricRepository) Save(machineID string, metric *domain.MetricPoint) error {
	query := `
		INSERT INTO metrics (
			machine_id, timestamp, cpu, memory_used, memory_total,
			status, net_rx, net_tx, disk_read, disk_write
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		machineID,
		metric.Timestamp.Unix(),
		metric.CPU,
		metric.MemoryUsed,
		metric.MemoryTotal,
		metric.Status,
		metric.NetRxRate,
		metric.NetTxRate,
		metric.DiskRead,
		metric.DiskWrite,
	)

	if err != nil {
		return fmt.Errorf("failed to save metric: %w", err)
	}

	return nil
}

// GetHistory retourne l'historique des métriques pour une période
func (r *MetricRepository) GetHistory(machineID string, duration time.Duration) ([]domain.MetricPoint, error) {
	since := time.Now().Add(-duration).Unix()

	query := `
		SELECT timestamp, cpu, memory_used, memory_total, status,
		       net_rx, net_tx, disk_read, disk_write
		FROM metrics
		WHERE machine_id = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Query(query, machineID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []domain.MetricPoint

	for rows.Next() {
		var timestamp int64
		var metric domain.MetricPoint

		err := rows.Scan(
			&timestamp,
			&metric.CPU,
			&metric.MemoryUsed,
			&metric.MemoryTotal,
			&metric.Status,
			&metric.NetRxRate,
			&metric.NetTxRate,
			&metric.DiskRead,
			&metric.DiskWrite,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		metric.Timestamp = time.Unix(timestamp, 0)
		metrics = append(metrics, metric)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return metrics, nil
}

// GetLatest retourne la dernière métrique enregistrée
func (r *MetricRepository) GetLatest(machineID string) (*domain.MetricPoint, error) {
	query := `
		SELECT timestamp, cpu, memory_used, memory_total, status,
		       net_rx, net_tx, disk_read, disk_write
		FROM metrics
		WHERE machine_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var timestamp int64
	var metric domain.MetricPoint

	err := r.db.QueryRow(query, machineID).Scan(
		&timestamp,
		&metric.CPU,
		&metric.MemoryUsed,
		&metric.MemoryTotal,
		&metric.Status,
		&metric.NetRxRate,
		&metric.NetTxRate,
		&metric.DiskRead,
		&metric.DiskWrite,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest metric: %w", err)
	}

	metric.Timestamp = time.Unix(timestamp, 0)
	return &metric, nil
}

// DeleteOlderThan supprime les métriques plus anciennes que la durée spécifiée
func (r *MetricRepository) DeleteOlderThan(duration time.Duration) error {
	cutoff := time.Now().Add(-duration).Unix()

	query := `DELETE FROM metrics WHERE timestamp < ?`

	result, err := r.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Deleted %d old metric records\n", rowsAffected)
	}

	return nil
}

// GetStorageSize retourne la taille de stockage utilisée
func (r *MetricRepository) GetStorageSize() (int64, error) {
	query := `SELECT COUNT(*) FROM metrics`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get storage size: %w", err)
	}

	return count, nil
}
