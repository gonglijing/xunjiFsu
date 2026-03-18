package driver

import (
	"path/filepath"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func BenchmarkNewPreparedExecution_ModbusTCP(b *testing.B) {
	tmpDir := b.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	b.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})
	if err := database.InitParamDBWithPath(paramPath); err != nil {
		b.Fatalf("InitParamDBWithPath failed: %v", err)
	}

	resourceID := int64(99)
	device := &models.Device{
		ID:            1,
		Name:          "dev-1",
		DriverType:    "modbus_tcp",
		ResourceID:    &resourceID,
		IPAddress:     "127.0.0.1",
		PortNum:       502,
		DeviceAddress: "1",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		prepared := NewPreparedExecution(device)
		if prepared == nil {
			b.Fatal("prepared execution is nil")
		}
	}
}
