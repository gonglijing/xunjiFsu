package service

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type DriverRuntimeReader interface {
	GetRuntime(id int64) (*driverpkg.DriverRuntime, error)
}

type DriverRuntimeManager interface {
	LoadDriverFromModel(driver *models.Driver, resourceID int64) error
	UnloadDriver(id int64) error
}

type DriverFileItem struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type DriverService struct {
	driverManager DriverRuntimeManager
	runtimeReader DriverRuntimeReader
	driversDir    string
}

func NewDriverService(driverManager DriverRuntimeManager, runtimeReader DriverRuntimeReader, driversDir string) *DriverService {
	return &DriverService{
		driverManager: driverManager,
		runtimeReader: runtimeReader,
		driversDir:    driversDir,
	}
}

func (s *DriverService) ListDrivers() ([]*models.Driver, error) {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		return nil, err
	}
	for _, drv := range drivers {
		s.LoadAndSyncDriverVersion(drv)
		s.EnrichDriverModel(drv)
	}
	return drivers, nil
}

func (s *DriverService) LoadDriver(id int64) (*models.Driver, error) {
	return database.GetDriverByID(id)
}

func (s *DriverService) CreateDriver(driver *models.Driver) (*models.Driver, error) {
	if driver == nil {
		return nil, nil
	}
	if err := NormalizeDriverInput(s.driversDir, driver); err != nil {
		return nil, err
	}
	if _, err := os.Stat(s.driverFilePath(driver.Name, driver.FilePath)); err != nil {
		return nil, err
	}

	s.LoadAndSyncDriverVersion(driver)
	id, err := database.CreateDriver(driver)
	if err != nil {
		return nil, err
	}
	driver.ID = id

	if driver.Enabled == 1 && s.driverManager != nil {
		if err := s.driverManager.LoadDriverFromModel(driver, 0); err != nil {
			return nil, err
		}
	}
	return driver, nil
}

func (s *DriverService) UpdateDriver(driver *models.Driver) (*models.Driver, error) {
	if driver == nil {
		return nil, nil
	}
	if err := NormalizeDriverInput(s.driversDir, driver); err != nil {
		return nil, err
	}
	if _, err := os.Stat(s.driverFilePath(driver.Name, driver.FilePath)); err != nil {
		return nil, err
	}

	s.LoadAndSyncDriverVersion(driver)
	if err := database.UpdateDriver(driver); err != nil {
		return nil, err
	}

	if s.driverManager != nil {
		if driver.Enabled == 1 {
			if err := s.driverManager.LoadDriverFromModel(driver, 0); err != nil {
				return nil, err
			}
		} else {
			_ = s.driverManager.UnloadDriver(driver.ID)
		}
	}

	return driver, nil
}

func (s *DriverService) DeleteDriver(id int64) error {
	drv, err := database.GetDriverByID(id)
	if err != nil {
		return err
	}
	_ = database.DeleteDriver(id)
	if s.driverManager != nil {
		_ = s.driverManager.UnloadDriver(id)
	}
	_ = os.Remove(s.driverFilePath(drv.Name, drv.FilePath))
	return nil
}

func (s *DriverService) ReloadDriver(id int64) (*driverpkg.DriverRuntime, error) {
	driverModel, err := database.GetDriverByID(id)
	if err != nil {
		return nil, err
	}
	if !s.EnsureDriverFileExists(driverModel) {
		return nil, os.ErrNotExist
	}
	if s.driverManager == nil {
		return nil, driverpkg.ErrDriverNotFound
	}
	if err := s.driverManager.LoadDriverFromModel(driverModel, 0); err != nil {
		return nil, err
	}
	s.LoadAndSyncDriverVersion(driverModel)
	return s.RuntimeResponse(driverModel.ID), nil
}

func (s *DriverService) RuntimeResponse(id int64) *driverpkg.DriverRuntime {
	if s.runtimeReader == nil {
		return unloadedDriverRuntime(id)
	}
	runtime, err := s.runtimeReader.GetRuntime(id)
	if err != nil {
		return unloadedDriverRuntime(id)
	}
	return runtime
}

func (s *DriverService) LoadDriverRuntime(id int64) (*driverpkg.DriverRuntime, error) {
	if _, err := database.GetDriverByID(id); err != nil {
		return nil, err
	}
	if s.runtimeReader == nil {
		return unloadedDriverRuntime(id), nil
	}
	runtime, err := s.runtimeReader.GetRuntime(id)
	if err != nil {
		if err == driverpkg.ErrDriverNotLoaded {
			return unloadedDriverRuntime(id), nil
		}
		return nil, err
	}
	return runtime, nil
}

func (s *DriverService) ListDriverRuntimes() []*driverpkg.DriverRuntime {
	if lister, ok := s.runtimeReader.(interface {
		ListRuntimes() []*driverpkg.DriverRuntime
	}); ok {
		return lister.ListRuntimes()
	}
	return []*driverpkg.DriverRuntime{}
}

func (s *DriverService) ListDriverFiles() ([]DriverFileItem, error) {
	return ListDriverWasmFiles(s.driversDir)
}

func (s *DriverService) EnsureDriverFileExists(driverModel *models.Driver) bool {
	path := s.driverFilePath(driverModel.Name, driverModel.FilePath)
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func NormalizeDriverInput(driversDir string, driver *models.Driver) error {
	if driver == nil {
		return sql.ErrNoRows
	}
	driver.Name = strings.TrimSpace(driver.Name)
	if driver.Name == "" {
		return sql.ErrNoRows
	}
	driver.FilePath = strings.TrimSpace(driver.FilePath)
	if driver.FilePath == "" {
		driver.FilePath = driverPath(driversDir, driver.Name, "")
	}
	if driver.Enabled != 1 {
		driver.Enabled = 0
	}
	return nil
}

func SaveDriverUploadFile(driversDir, filename string, source io.Reader) (string, error) {
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		return "", err
	}

	destPath := filepath.Join(driversDir, filename)
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, source); err != nil {
		return "", err
	}

	return destPath, nil
}

func ListDriverWasmFiles(driversDir string) ([]DriverFileItem, error) {
	entries, err := os.ReadDir(driversDir)
	if err != nil {
		return nil, err
	}

	files := make([]DriverFileItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !IsWasmFileName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, DriverFileItem{
			Name:     entry.Name(),
			Size:     info.Size(),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return files, nil
}

func unloadedDriverRuntime(id int64) *driverpkg.DriverRuntime {
	return &driverpkg.DriverRuntime{
		ID:     id,
		Loaded: false,
	}
}

func IsWasmFileName(filename string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(filename)), ".wasm")
}

func (s *DriverService) EnrichDriverModel(driver *models.Driver) {
	if driver == nil {
		return
	}
	path := s.driverFilePath(driver.Name, driver.FilePath)
	if info, err := os.Stat(path); err == nil {
		driver.Size = info.Size()
		driver.Filename = filepath.Base(path)
	}
	if s.runtimeReader != nil {
		if runtime, err := s.runtimeReader.GetRuntime(driver.ID); err == nil && runtime != nil {
			driver.Loaded = runtime.Loaded
			driver.ResourceID = runtime.ResourceID
			driver.LastActive = runtime.LastActive
			driver.Exports = runtime.ExportedFunctions
		}
	}
}

func (s *DriverService) LoadAndSyncDriverVersion(driver *models.Driver) {
	if driver == nil {
		return
	}
	wasmPath := s.driverFilePath(driver.Name, driver.FilePath)
	wasmData, err := os.ReadFile(wasmPath)
	if err != nil {
		return
	}

	version, productKey, err := driverpkg.ExtractDriverMetadata(wasmData)
	if err != nil {
		return
	}
	if version != "" {
		driver.Version = version
		if driver.ID > 0 {
			_ = database.UpdateDriverVersion(driver.ID, version)
		}
	}
	if productKey != "" {
		driver.ProductKey = productKey
	}
}

type DriverUploadResult struct {
	Filename   string `json:"filename"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Version    string `json:"version"`
	ProductKey string `json:"product_key"`
}

type DriverDownloadFile struct {
	Driver   *models.Driver
	Path     string
	Size     int64
	Name     string
	OpenFile *os.File
}

func (s *DriverService) SaveUploadedDriver(filename string, size int64, source io.Reader) (*DriverUploadResult, error) {
	destPath, err := SaveDriverUploadFile(s.driversDir, filename, source)
	if err != nil {
		return nil, err
	}

	driverName := strings.TrimSuffix(filename, ".wasm")
	_ = database.UpsertDriverFile(driverName, destPath)

	result := &DriverUploadResult{
		Filename: filename,
		Path:     destPath,
		Size:     size,
	}

	if wasmData, err := os.ReadFile(destPath); err == nil {
		if version, productKey, err := driverpkg.ExtractDriverMetadata(wasmData); err == nil {
			if version != "" {
				result.Version = version
				_ = database.UpdateDriverVersionByName(driverName, version)
			}
			if productKey != "" {
				result.ProductKey = productKey
			}
		}
	}

	return result, nil
}

func (s *DriverService) OpenDriverDownload(id int64) (*DriverDownloadFile, error) {
	driverModel, err := database.GetDriverByID(id)
	if err != nil {
		return nil, err
	}

	filePath := s.driverFilePath(driverModel.Name, driverModel.FilePath)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	return &DriverDownloadFile{
		Driver:   driverModel,
		Path:     filePath,
		Size:     info.Size(),
		Name:     filepath.Base(filePath),
		OpenFile: file,
	}, nil
}

func (s *DriverService) driverFilePath(driverName, filePath string) string {
	return driverPath(s.driversDir, driverName, filePath)
}

func driverPath(driversDir, driverName, filePath string) string {
	if filePath != "" {
		return filePath
	}
	dir := strings.TrimSpace(driversDir)
	if dir == "" {
		dir = "drivers"
	}
	return filepath.Join(dir, driverName+".wasm")
}
