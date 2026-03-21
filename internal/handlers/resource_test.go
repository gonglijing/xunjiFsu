package handlers

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/httpapi"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func TestUpdateResourceRefreshesDriverExecutorCache(t *testing.T) {
	tmpDir := t.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	t.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})

	if err := database.InitParamDBWithPath(paramPath); err != nil {
		t.Fatalf("InitParamDBWithPath failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		path TEXT,
		enabled INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create resources table failed: %v", err)
	}

	resourceID, err := database.AddResource(&models.Resource{
		Name:    "r1",
		Type:    "net",
		Path:    "127.0.0.1:502",
		Enabled: 1,
	})
	if err != nil {
		t.Fatalf("AddResource failed: %v", err)
	}

	executor := driver.NewDriverExecutor(driver.NewDriverManager())
	c1, c2 := net.Pipe()
	defer c2.Close()

	executor.RegisterTCP(resourceID, c1)
	executor.SetResourcePath(resourceID, "127.0.0.1:502")
	api := httpapi.NewResourceAPI(service.NewResourceService(executor))

	req := httptest.NewRequest(http.MethodPut, "/resources/"+strconv.FormatInt(resourceID, 10), strings.NewReader(`{"name":"r1","type":"net","path":"127.0.0.1:503","enabled":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.FormatInt(resourceID, 10))
	w := httptest.NewRecorder()

	api.UpdateResource(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := executor.GetResourcePath(resourceID); got != "127.0.0.1:503" {
		t.Fatalf("resource path = %q, want %q", got, "127.0.0.1:503")
	}
	if _, err := c1.Write([]byte("x")); err == nil {
		t.Fatalf("expected closed conn after resource path update")
	}
}
