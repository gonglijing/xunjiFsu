package database

import (
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
		{"timeout", "INTEGER DEFAULT 1000"},
		{"driver_id", "INTEGER"},
		{"resource_id", "INTEGER"},
	}

	for _, col := range columns {
		ParamDB.Exec(fmt.Sprintf("ALTER TABLE devices ADD COLUMN %s %s", col.name, col.ctype))
	}

	return nil
}

// CreateDevice 创建设备
func CreateDevice(device *models.Device) (int64, error) {
	result, err := ParamDB.Exec(
		`INSERT INTO devices (name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity, 
			ip_address, port_num, device_address, collect_interval, timeout, driver_id, enabled, resource_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		device.Name, device.Description, device.ProductKey, device.DeviceKey, device.DriverType, device.SerialPort, device.BaudRate, device.DataBits,
		device.StopBits, device.Parity, device.IPAddress, device.PortNum, device.DeviceAddress,
		device.CollectInterval, device.Timeout, device.DriverID, device.Enabled, device.ResourceID,
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
			ip_address, port_num, device_address, collect_interval, timeout, driver_id, enabled, resource_id, created_at, updated_at 
		FROM devices WHERE id = ?`,
		id,
	).Scan(&device.ID, &device.Name, &device.Description, &device.ProductKey, &device.DeviceKey, &device.DriverType, &device.SerialPort, &device.BaudRate,
		&device.DataBits, &device.StopBits, &device.Parity, &device.IPAddress, &device.PortNum,
		&device.DeviceAddress, &device.CollectInterval, &device.Timeout, &device.DriverID, &device.Enabled, &device.ResourceID,
		&device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// GetAllDevices 获取所有设备
func GetAllDevices() ([]*models.Device, error) {
	rows, err := ParamDB.Query(
		`SELECT id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity, 
			ip_address, port_num, device_address, collect_interval, timeout, driver_id, enabled, resource_id, created_at, updated_at 
		FROM devices ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.ProductKey, &device.DeviceKey, &device.DriverType, &device.SerialPort,
			&device.BaudRate, &device.DataBits, &device.StopBits, &device.Parity, &device.IPAddress, &device.PortNum,
			&device.DeviceAddress, &device.CollectInterval, &device.Timeout, &device.DriverID, &device.Enabled, &device.ResourceID,
			&device.CreatedAt, &device.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, nil
}

// UpdateDevice 更新设备
func UpdateDevice(device *models.Device) error {
	_, err := ParamDB.Exec(
		`UPDATE devices SET name = ?, description = ?, product_key = ?, device_key = ?, driver_type = ?, serial_port = ?, baud_rate = ?, 
			data_bits = ?, stop_bits = ?, parity = ?, ip_address = ?, port_num = ?, 
			device_address = ?, collect_interval = ?, timeout = ?, driver_id = ?, enabled = ?, resource_id = ?, 
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		device.Name, device.Description, device.ProductKey, device.DeviceKey, device.DriverType, device.SerialPort, device.BaudRate, device.DataBits,
		device.StopBits, device.Parity, device.IPAddress, device.PortNum,
		device.DeviceAddress, device.CollectInterval, device.Timeout, device.DriverID, device.Enabled, device.ResourceID,
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
			device_address = ?, collect_interval = ?, timeout = ?, driver_id = ?, enabled = ?, resource_id = ?, 
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		device.Name, device.Description, device.ProductKey, device.DeviceKey, device.DriverType, device.SerialPort, device.BaudRate, device.DataBits,
		device.StopBits, device.Parity, device.IPAddress, device.PortNum,
		device.DeviceAddress, device.CollectInterval, device.Timeout, device.DriverID, device.Enabled, device.ResourceID,
		id,
	)
	return err
}
