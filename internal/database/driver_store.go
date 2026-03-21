package database

import (
	"database/sql"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

const selectDriverFields = `SELECT id, name, file_path, description, version, config_schema, enabled, created_at, updated_at FROM drivers`

// ==================== 驱动操作 (param.db - 直接写) ====================

// CreateDriver 创建驱动
func CreateDriver(driver *models.Driver) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO drivers (name, file_path, description, version, config_schema, enabled) VALUES (?, ?, ?, ?, ?, ?)",
		driver.Name, driver.FilePath, driver.Description, driver.Version, driver.ConfigSchema, driver.Enabled,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// LoadDriver 根据ID获取驱动
func LoadDriver(id int64) (*models.Driver, error) {
	return loadDriver(selectDriverFields+" WHERE id = ?", id)
}

// ListDrivers 获取所有驱动
func ListDrivers() ([]*models.Driver, error) {
	return listDrivers(selectDriverFields+" ORDER BY id", nil)
}

// GetDriverByName 根据名称获取驱动
func GetDriverByName(name string) (*models.Driver, error) {
	return loadDriver(selectDriverFields+" WHERE name = ?", name)
}

type driverScanner interface {
	Scan(dest ...any) error
}

func loadDriver(query string, args ...any) (*models.Driver, error) {
	driver := &models.Driver{}
	err := scanDriver(ParamDB.QueryRow(query, args...), driver)
	if err != nil {
		return nil, err
	}
	return driver, nil
}

func listDrivers(query string, args []any) ([]*models.Driver, error) {
	return queryList[*models.Driver](ParamDB,
		query,
		args,
		func(rows *sql.Rows) (*models.Driver, error) {
			driver := &models.Driver{}
			if err := scanDriver(rows, driver); err != nil {
				return nil, err
			}
			return driver, nil
		},
	)
}

func scanDriver(scanner driverScanner, driver *models.Driver) error {
	return scanner.Scan(
		&driver.ID,
		&driver.Name,
		&driver.FilePath,
		&driver.Description,
		&driver.Version,
		&driver.ConfigSchema,
		&driver.Enabled,
		&driver.CreatedAt,
		&driver.UpdatedAt,
	)
}

// UpdateDriver 更新驱动
func UpdateDriver(driver *models.Driver) error {
	_, err := ParamDB.Exec(
		"UPDATE drivers SET name = ?, file_path = ?, description = ?, version = ?, config_schema = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		driver.Name, driver.FilePath, driver.Description, driver.Version, driver.ConfigSchema, driver.Enabled, driver.ID,
	)
	return err
}

// DeleteDriver 删除驱动
func DeleteDriver(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM drivers WHERE id = ?", id)
	return err
}

// UpsertDriverFile 保存或忽略重复的驱动记录
func UpsertDriverFile(name, path string) error {
	_, err := ParamDB.Exec(
		`INSERT OR IGNORE INTO drivers (name, file_path, description, version, config_schema, enabled) 
		 VALUES (?, ?, '', '', '', 1)`, name, path)
	return err
}

// UpdateDriverVersionByName updates driver version by name.
func UpdateDriverVersionByName(name, version string) error {
	_, err := ParamDB.Exec(
		"UPDATE drivers SET version = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?",
		version, name,
	)
	return err
}

// UpdateDriverVersion updates driver version by id.
func UpdateDriverVersion(id int64, version string) error {
	_, err := ParamDB.Exec(
		"UPDATE drivers SET version = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		version, id,
	)
	return err
}
