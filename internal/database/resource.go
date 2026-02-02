package database

import (
	"database/sql"
	"fmt"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// InitResourceTable 创建资源表
func InitResourceTable() error {
	_, err := ParamDB.Exec(`CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		path TEXT,
		enabled INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	return ensureResourcePathColumn()
}

func AddResource(r *models.Resource) (int64, error) {
	res, err := ParamDB.Exec(`INSERT INTO resources (name, type, path, enabled) VALUES (?,?,?,?)`, r.Name, r.Type, r.Path, r.Enabled)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListResources() ([]*models.Resource, error) {
	rows, err := ParamDB.Query(`SELECT id, name, type, path, enabled, created_at, updated_at FROM resources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Resource
	for rows.Next() {
		r := &models.Resource{}
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Path, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func UpdateResource(r *models.Resource) error {
	_, err := ParamDB.Exec(`UPDATE resources SET name=?, type=?, path=?, enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, r.Name, r.Type, r.Path, r.Enabled, r.ID)
	return err
}

func DeleteResource(id int64) error {
	_, err := ParamDB.Exec(`DELETE FROM resources WHERE id=?`, id)
	return err
}

func ToggleResource(id int64, enabled int) error {
	_, err := ParamDB.Exec(`UPDATE resources SET enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, enabled, id)
	return err
}

// EnsureDeviceResourceColumn adds resource_id column if missing.
func EnsureDeviceResourceColumn() {
	ParamDB.Exec(`ALTER TABLE devices ADD COLUMN resource_id INTEGER REFERENCES resources(id)`)
}

func BindDeviceResource(deviceID, resourceID int64) error {
	_, err := ParamDB.Exec(`UPDATE devices SET resource_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resourceID, deviceID)
	return err
}

// GetResourceByID returns resource
func GetResourceByID(id int64) (*models.Resource, error) {
	r := &models.Resource{}
	err := ParamDB.QueryRow(`SELECT id,name,type,path,enabled,created_at,updated_at FROM resources WHERE id=?`, id).
		Scan(&r.ID, &r.Name, &r.Type, &r.Path, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// ensureResourcePathColumn adds path column if older schema is missing it,
// and backfills from legacy port column when possible.
func ensureResourcePathColumn() error {
	hasPath, err := columnExists(ParamDB, "resources", "path")
	if err != nil {
		return err
	}
	if !hasPath {
		if _, err := ParamDB.Exec(`ALTER TABLE resources ADD COLUMN path TEXT`); err != nil {
			return fmt.Errorf("add path column: %w", err)
		}
		// 尝试用旧 schema 的 port 字段回填
		_, _ = ParamDB.Exec(`UPDATE resources SET path = COALESCE(path, port, '')`)
	}
	return nil
}

// columnExists checks if a column exists in a table.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var (
		cid     int
		name    string
		typ     string
		notnull int
		dflt    sql.NullString
		pk      int
	)
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, nil
}
