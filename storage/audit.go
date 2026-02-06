package storage

import (
	"log"
	"time"
)

// LogAction enregistre une action utilisateur
func (db *DB) LogAction(user, action, target, details, ip string) error {
	query := `INSERT INTO audit_logs (timestamp, user, action, target, details, ip_address) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, time.Now(), user, action, target, details, ip)
	if err != nil {
		log.Printf("Erreur audit log: %v", err)
	}
	return err
}

// GetAuditLogs récupère les logs d'audit récents
func (db *DB) GetAuditLogs(limit int) ([]AuditLog, error) {
	query := `SELECT id, timestamp, user, action, target, details, ip_address 
			  FROM audit_logs ORDER BY timestamp DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.User, &l.Action, &l.Target, &l.Details, &l.IPAddress); err != nil {
			log.Printf("Erreur scan audit: %v", err)
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}
