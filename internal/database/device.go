package database

import (
	"database/sql"
	"fmt"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// InitDeviceTable 初始化设备表
func InitDeviceTable() error {
	// 只添加缺失的列
	columns := []struct {
		name  string
		ctype string
	}{
		{"product_key", "TEXT"},
		{"device_key", "TEXT"},
		{"driver_type", "TEXT DEFAULT 'modbus_rtu'"},
		{"serial_port", "TEXT"},
		{"baud_rate", "INTEGER DEFAULT 9600"},
		{"data_bits", "INTEGER DEFAULT 8"},
		{"stop_bits", "INTEGER DEFAULT 1"},
		{"parity", "TEXT CHECK(parity IN ('N', 'O', 'E'))"},
		{"ip_address", "TEXT"},
		{"port_num", "INTEGER DEFAULT 502"},
		{"device_address", "TEXT"},
		{"collect_interval", "INTEGER DEFAULT 5000"},
		{"storage_interval", "INTEGER DEFAULT 300"},
		{"timeout", "INTEGER DEFAULT 1000"},
		{"driver_id", "INTEGER"},
		{"resource_id", "INTEGER"},
	}

	for _, col := range columns {
		ParamDB.Exec(fmt.Sprintf("ALTER TABLE devices ADD COLUMN %s %s", col.name, col.ctype))
	}

	return cleanupLegacyDeviceColumns()
}

func cleanupLegacyDeviceColumns() error {
	hasDeviceConfig, err := columnExists(ParamDB, "devices", "device_config")
	if err != nil {
		return err
	}
	hasUploadInterval, err := columnExists(ParamDB, "devices", "upload_interval")
	if err != nil {
		return err
	}
	hasProtocol, err := columnExists(ParamDB, "devices", "protocol")
	if err != nil {
		return err
	}
	if !hasDeviceConfig && !hasUploadInterval && !hasProtocol {
		return nil
	}

	tx, err := ParamDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`CREATE TABLE devices_new (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		product_key TEXT,
		device_key TEXT,
		driver_type TEXT DEFAULT 'modbus_rtu',
		serial_port TEXT,
		resource_id INTEGER,
		driver_id INTEGER,
		collect_interval INTEGER DEFAULT 5000,
		storage_interval INTEGER DEFAULT 300,
		timeout INTEGER DEFAULT 1000,
		baud_rate INTEGER DEFAULT 9600,
		data_bits INTEGER DEFAULT 8,
		stop_bits INTEGER DEFAULT 1,
		parity TEXT DEFAULT 'N' CHECK(parity IN ('N', 'O', 'E')),
		ip_address TEXT,
		port_num INTEGER DEFAULT 502,
		device_address TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE SET NULL,
		FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE SET NULL
	)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO devices_new (
		id, name, description, product_key, device_key, driver_type,
		serial_port, resource_id, driver_id,
		collect_interval, storage_interval, timeout,
		baud_rate, data_bits, stop_bits, parity,
		ip_address, port_num, device_address,
		enabled, created_at, updated_at
	)
	SELECT
		id, name, description, product_key, device_key, COALESCE(driver_type, 'modbus_rtu'),
		serial_port, resource_id, driver_id,
		COALESCE(collect_interval, 5000), COALESCE(storage_interval, 300), COALESCE(timeout, 1000),
		COALESCE(baud_rate, 9600), COALESCE(data_bits, 8), COALESCE(stop_bits, 1), COALESCE(parity, 'N'),
		ip_address, COALESCE(port_num, 502), device_address,
		COALESCE(enabled, 1), created_at, updated_at
	FROM devices`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE devices`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`ALTER TABLE devices_new RENAME TO devices`)
	if err != nil {
		return err
	}

	if _, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_resource ON devices(resource_id)`); err != nil {
		return err
	}
	if _, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_driver ON devices(driver_id)`); err != nil {
		return err
	}

	return tx.Commit()
}

// CreateDevice 创建设备
func CreateDevice(device *models.Device) (int64, error) {
	result, err := ParamDB.Exec(
		`INSERT INTO devices (name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity, 
			ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		device.Name, device.Description, device.ProductKey, device.DeviceKey, device.DriverType, device.SerialPort, device.BaudRate, device.DataBits,
		device.StopBits, device.Parity, device.IPAddress, device.PortNum, device.DeviceAddress,
		device.CollectInterval, device.StorageInterval, device.Timeout, device.DriverID, device.Enabled, device.ResourceID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetDeviceByID 根据ID获取设备
func GetDeviceByID(id int64) (*models.Device, error) {
	device := &models.Device{}
	err := ParamDB.QueryRow(
		`SELECT id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity, 
			ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id, created_at, updated_at 
		FROM devices WHERE id = ?`,
		id,
	).Scan(&device.ID, &device.Name, &device.Description, &device.ProductKey, &device.DeviceKey, &device.DriverType, &device.SerialPort, &device.BaudRate,
		&device.DataBits, &device.StopBits, &device.Parity, &device.IPAddress, &device.PortNum,
		&device.DeviceAddress, &device.CollectInterval, &device.StorageInterval, &device.Timeout, &device.DriverID, &device.Enabled, &device.ResourceID,
		&device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// GetDeviceByIdentity 按 product_key/device_key 获取设备
func GetDeviceByIdentity(productKey, deviceKey string) (*models.Device, error) {
	device := &models.Device{}
	err := ParamDB.QueryRow(
		`SELECT id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity, 
			ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id, created_at, updated_at 
		FROM devices WHERE product_key = ? AND device_key = ? LIMIT 1`,
		productKey, deviceKey,
	).Scan(&device.ID, &device.Name, &device.Description, &device.ProductKey, &device.DeviceKey, &device.DriverType, &device.SerialPort, &device.BaudRate,
		&device.DataBits, &device.StopBits, &device.Parity, &device.IPAddress, &device.PortNum,
		&device.DeviceAddress, &device.CollectInterval, &device.StorageInterval, &device.Timeout, &device.DriverID, &device.Enabled, &device.ResourceID,
		&device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// GetAllDevices 获取所有设备
func GetAllDevices() ([]*models.Device, error) {
	return queryList[*models.Device](ParamDB,
		`SELECT id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity, 
			ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id, created_at, updated_at 
		FROM devices ORDER BY id`,
		nil,
		func(rows *sql.Rows) (*models.Device, error) {
			device := &models.Device{}
			if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.ProductKey, &device.DeviceKey, &device.DriverType, &device.SerialPort,
				&device.BaudRate, &device.DataBits, &device.StopBits, &device.Parity, &device.IPAddress, &device.PortNum,
				&device.DeviceAddress, &device.CollectInterval, &device.StorageInterval, &device.Timeout, &device.DriverID, &device.Enabled, &device.ResourceID,
				&device.CreatedAt, &device.UpdatedAt); err != nil {
				return nil, err
			}
			return device, nil
		},
	)
}

// UpdateDevice 更新设备
func UpdateDevice(device *models.Device) error {
	_, err := ParamDB.Exec(
		`UPDATE devices SET name = ?, description = ?, product_key = ?, device_key = ?, driver_type = ?, serial_port = ?, baud_rate = ?, 
			data_bits = ?, stop_bits = ?, parity = ?, ip_address = ?, port_num = ?, 
			device_address = ?, collect_interval = ?, storage_interval = ?, timeout = ?, driver_id = ?, enabled = ?, resource_id = ?, 
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		device.Name, device.Description, device.ProductKey, device.DeviceKey, device.DriverType, device.SerialPort, device.BaudRate, device.DataBits,
		device.StopBits, device.Parity, device.IPAddress, device.PortNum,
		device.DeviceAddress, device.CollectInterval, device.StorageInterval, device.Timeout, device.DriverID, device.Enabled, device.ResourceID,
		device.ID,
	)
	return err
}

// DeleteDevice 删除设备
func DeleteDevice(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM devices WHERE id = ?", id)
	return err
}

// ToggleDevice 切换设备状态
func ToggleDevice(id int64) error {
	_, err := ParamDB.Exec("UPDATE devices SET enabled = 1 - enabled, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

// UpdateDeviceEnabled 更新设备使能状态
func UpdateDeviceEnabled(id int64, enabled int) error {
	_, err := ParamDB.Exec("UPDATE devices SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", enabled, id)
	return err
}

// UpdateDeviceWithID 根据ID更新设备信息（用于API）
func UpdateDeviceWithID(id int64, device *models.Device) error {
	_, err := ParamDB.Exec(
		`UPDATE devices SET name = ?, description = ?, product_key = ?, device_key = ?, driver_type = ?, serial_port = ?, baud_rate = ?, 
			data_bits = ?, stop_bits = ?, parity = ?, ip_address = ?, port_num = ?, 
			device_address = ?, collect_interval = ?, storage_interval = ?, timeout = ?, driver_id = ?, enabled = ?, resource_id = ?, 
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		device.Name, device.Description, device.ProductKey, device.DeviceKey, device.DriverType, device.SerialPort, device.BaudRate, device.DataBits,
		device.StopBits, device.Parity, device.IPAddress, device.PortNum,
		device.DeviceAddress, device.CollectInterval, device.StorageInterval, device.Timeout, device.DriverID, device.Enabled, device.ResourceID,
		id,
	)
	return err
}

// UpdateDeviceDriverID 更新设备驱动ID
func UpdateDeviceDriverID(deviceID int64, driverID int64) error {
	_, err := ParamDB.Exec(
		"UPDATE devices SET driver_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		driverID, deviceID,
	)
	return err
}
