package database

import "fmt"

const defaultRuntimeAuditLimit = 20
const maxRuntimeAuditLimit = 200

type RuntimeConfigAudit struct {
	ID               int64  `json:"id" db:"id"`
	OperatorUserID   int64  `json:"operator_user_id" db:"operator_user_id"`
	OperatorUsername string `json:"operator_username" db:"operator_username"`
	SourceIP         string `json:"source_ip" db:"source_ip"`
	Changes          string `json:"changes" db:"changes"`
	CreatedAt        string `json:"created_at" db:"created_at"`
}

func InitRuntimeConfigAuditTable() error {
	if ParamDB == nil {
		return fmt.Errorf("param database is not initialized")
	}

	if _, err := ParamDB.Exec(`CREATE TABLE IF NOT EXISTS runtime_config_audits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		operator_user_id INTEGER DEFAULT 0,
		operator_username TEXT DEFAULT 'unknown',
		source_ip TEXT DEFAULT '',
		changes TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	if _, err := ParamDB.Exec(`CREATE INDEX IF NOT EXISTS idx_runtime_config_audits_created_at ON runtime_config_audits(created_at DESC)`); err != nil {
		return err
	}

	return nil
}

func CreateRuntimeConfigAudit(audit *RuntimeConfigAudit) (int64, error) {
	if audit == nil {
		return 0, fmt.Errorf("runtime config audit is nil")
	}

	result, err := ParamDB.Exec(`INSERT INTO runtime_config_audits (
		operator_user_id, operator_username, source_ip, changes
	) VALUES (?, ?, ?, ?)`,
		audit.OperatorUserID,
		audit.OperatorUsername,
		audit.SourceIP,
		audit.Changes,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func ListRuntimeConfigAudits(limit int) ([]*RuntimeConfigAudit, error) {
	if limit <= 0 {
		limit = defaultRuntimeAuditLimit
	}
	if limit > maxRuntimeAuditLimit {
		limit = maxRuntimeAuditLimit
	}

	rows, err := ParamDB.Query(`SELECT
		id,
		COALESCE(operator_user_id, 0),
		COALESCE(operator_username, 'unknown'),
		COALESCE(source_ip, ''),
		COALESCE(changes, '{}'),
		created_at
	FROM runtime_config_audits
	ORDER BY id DESC
	LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*RuntimeConfigAudit, 0)
	for rows.Next() {
		item := &RuntimeConfigAudit{}
		if err := rows.Scan(&item.ID, &item.OperatorUserID, &item.OperatorUsername, &item.SourceIP, &item.Changes, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
