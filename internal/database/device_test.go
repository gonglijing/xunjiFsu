package database

import (
	"path/filepath"
	"testing"
)

func setupDeviceTestDB(t *testing.T) {
	t.Helper()

	originalParamDB := ParamDB
	t.Cleanup(func() {
		if ParamDB != nil {
			_ = ParamDB.Close()
		}
		ParamDB = originalParamDB
	})

	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	tmpDir := t.TempDir()
	var err error
	ParamDB, err = openSQLite(filepath.Join(tmpDir, "param.db"), 1, 1)
	if err != nil {
		t.Fatalf("open param db: %v", err)
	}

	_, err = ParamDB.Exec(`CREATE TABLE drivers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		file_path TEXT NOT NULL,
		description TEXT,
		version TEXT,
		config_schema TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create drivers: %v", err)
	}

	_, err = ParamDB.Exec(`CREATE TABLE resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		type TEXT NOT NULL,
		path TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create resources: %v", err)
	}

	_, err = ParamDB.Exec(`CREATE TABLE devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		product_key TEXT,
		device_key TEXT,
		driver_type TEXT DEFAULT 'modbus_rtu',
		serial_port TEXT,
		resource_id INTEGER,
		driver_id INTEGER,
		device_config TEXT,
		collect_interval INTEGER DEFAULT 5000,
		upload_interval INTEGER DEFAULT 10000,
		timeout INTEGER DEFAULT 1000,
		baud_rate INTEGER DEFAULT 9600,
		data_bits INTEGER DEFAULT 8,
		stop_bits INTEGER DEFAULT 1,
		parity TEXT DEFAULT 'N',
		ip_address TEXT,
		port_num INTEGER,
		device_address TEXT,
		protocol TEXT DEFAULT 'tcp',
		storage_interval INTEGER DEFAULT 300,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create legacy devices: %v", err)
	}
}

func TestInitDeviceTable_CleansLegacyColumns(t *testing.T) {
	setupDeviceTestDB(t)

	_, err := ParamDB.Exec(`INSERT INTO devices (
		name, description, product_key, device_key, driver_type,
		serial_port, resource_id, driver_id,
		device_config, collect_interval, upload_interval, timeout,
		baud_rate, data_bits, stop_bits, parity,
		ip_address, port_num, device_address, protocol,
		storage_interval, enabled
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"D1", "desc", "pk", "dk", "modbus_tcp",
		"/dev/ttyUSB0", nil, nil,
		"{\"x\":1}", 2000, 8000, 1500,
		19200, 7, 2, "E",
		"192.168.1.20", 1502, "2", "udp",
		600, 1,
	)
	if err != nil {
		t.Fatalf("insert legacy device: %v", err)
	}

	if err := InitDeviceTable(); err != nil {
		t.Fatalf("InitDeviceTable: %v", err)
	}

	hasDeviceConfig, err := columnExists(ParamDB, "devices", "device_config")
	if err != nil {
		t.Fatalf("columnExists device_config: %v", err)
	}
	hasUploadInterval, err := columnExists(ParamDB, "devices", "upload_interval")
	if err != nil {
		t.Fatalf("columnExists upload_interval: %v", err)
	}
	hasProtocol, err := columnExists(ParamDB, "devices", "protocol")
	if err != nil {
		t.Fatalf("columnExists protocol: %v", err)
	}
	if hasDeviceConfig || hasUploadInterval || hasProtocol {
		t.Fatalf("expected legacy columns removed, got device_config=%v upload_interval=%v protocol=%v", hasDeviceConfig, hasUploadInterval, hasProtocol)
	}

	device, err := GetDeviceByID(1)
	if err != nil {
		t.Fatalf("GetDeviceByID: %v", err)
	}
	if device.Name != "D1" || device.DriverType != "modbus_tcp" {
		t.Fatalf("unexpected device core fields: %+v", device)
	}
	if device.PortNum != 1502 || device.StorageInterval != 600 || device.Timeout != 1500 {
		t.Fatalf("unexpected device numeric fields: %+v", device)
	}
	if device.Parity != "E" || device.DeviceAddress != "2" {
		t.Fatalf("unexpected device protocol fields: %+v", device)
	}
}
