package adapters

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type pandaXSyncPayload struct {
	Timestamp  int64                 `json:"ts"`
	SubDevices []pandaXSyncSubDevice `json:"subDevices"`
}

type pandaXSyncSubDevice struct {
	ProductKey string                 `json:"productKey"`
	DeviceName string                 `json:"deviceName"`
	Timestamp  int64                  `json:"ts"`
	Values     map[string]any `json:"values"`
}

// SyncDevices 触发子设备及遥测模型同步（仅手动触发，不在启动时自动执行）
func (a *PandaXAdapter) SyncDevices() error {
	if !a.isInitialized() {
		return fmt.Errorf("adapter not initialized")
	}

	devices, err := database.ListDevices()
	if err != nil {
		return fmt.Errorf("获取设备列表失败: %w", err)
	}

	latestData, err := database.GetAllDevicesLatestData()
	if err != nil {
		return fmt.Errorf("获取设备最新遥测失败: %w", err)
	}

	topic, body, count, err := a.buildSyncDevicesPayload(devices, latestData)
	if err != nil {
		return err
	}
	if err := a.publish(topic, body); err != nil {
		return fmt.Errorf("发布设备同步消息失败: %w", err)
	}

	slog.Info("PandaX device sync published", "adapter", a.name, "topic", topic, "devices", count)
	return nil
}

func (a *PandaXAdapter) buildSyncDevicesPayload(devices []*models.Device, latestData []*database.LatestDeviceData) (string, []byte, int, error) {
	topic := a.syncRegisterTopic()
	nowMS := time.Now().UnixMilli()
	subDevices := a.buildSyncSubDevices(devices, latestData, nowMS)
	if err := validateSyncSubDevices(subDevices); err != nil {
		return topic, nil, len(subDevices), err
	}

	body, _ := json.Marshal(pandaXSyncPayload{
		Timestamp:  nowMS,
		SubDevices: subDevices,
	})
	return topic, body, len(subDevices), nil
}

func (a *PandaXAdapter) syncRegisterTopic() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.gatewayRegisterTopic
}

func (a *PandaXAdapter) buildSyncSubDevices(devices []*models.Device, latestData []*database.LatestDeviceData, nowMS int64) []pandaXSyncSubDevice {
	latestByDeviceID := latestDeviceDataByID(latestData)
	productKeyByDeviceID := resolveSyncProductKeyByDeviceID(devices)
	subDevices := make([]pandaXSyncSubDevice, 0, len(devices))

	for _, dev := range devices {
		if dev == nil || dev.ID <= 0 {
			continue
		}
		subDevice := a.buildSyncSubDevice(dev, latestByDeviceID[dev.ID], productKeyByDeviceID, nowMS)
		subDevices = append(subDevices, subDevice)
	}

	sortSyncSubDevices(subDevices)
	return subDevices
}

func latestDeviceDataByID(latestData []*database.LatestDeviceData) map[int64]*database.LatestDeviceData {
	latestByDeviceID := make(map[int64]*database.LatestDeviceData, len(latestData))
	for _, item := range latestData {
		if item == nil || item.DeviceID <= 0 || item.DeviceID == models.SystemStatsDeviceID {
			continue
		}
		latestByDeviceID[item.DeviceID] = item
	}
	return latestByDeviceID
}

func (a *PandaXAdapter) buildSyncSubDevice(
	dev *models.Device,
	latest *database.LatestDeviceData,
	productKeyByDeviceID map[int64]string,
	nowMS int64,
) pandaXSyncSubDevice {
	a.backfillResolvedProductKey(dev, productKeyByDeviceID)
	collectData := buildSyncCollectData(dev, latest, productKeyByDeviceID)

	return pandaXSyncSubDevice{
		ProductKey: strings.TrimSpace(collectData.ProductKey),
		DeviceName: pickFirstNonEmpty(collectData.DeviceName, strings.TrimSpace(dev.Name)),
		Timestamp:  syncTimestampOrNow(collectData.Timestamp, nowMS),
		Values:     buildSyncValues(collectData.Fields),
	}
}

func (a *PandaXAdapter) backfillResolvedProductKey(dev *models.Device, productKeyByDeviceID map[int64]string) {
	if dev == nil {
		return
	}

	resolvedProductKey := strings.TrimSpace(productKeyByDeviceID[dev.ID])
	if resolvedProductKey == "" || strings.TrimSpace(dev.ProductKey) == resolvedProductKey {
		return
	}

	if err := database.UpdateDeviceProductKey(dev.ID, resolvedProductKey); err != nil {
		slog.Info("PandaX device sync product key backfill failed",
			"adapter", a.name,
			"device_id", dev.ID,
			"product_key", resolvedProductKey,
			"error", err)
		return
	}
	dev.ProductKey = resolvedProductKey
}

func buildSyncCollectData(
	dev *models.Device,
	latest *database.LatestDeviceData,
	productKeyByDeviceID map[int64]string,
) *models.CollectData {
	collectData := &models.CollectData{
		DeviceID:   dev.ID,
		DeviceName: strings.TrimSpace(dev.Name),
		ProductKey: resolveSyncSubDeviceProductKey(dev, productKeyByDeviceID),
		DeviceKey:  strings.TrimSpace(dev.DeviceKey),
		Timestamp:  time.Now(),
		Fields:     map[string]string{},
	}

	if latest == nil {
		return collectData
	}
	if strings.TrimSpace(latest.DeviceName) != "" {
		collectData.DeviceName = strings.TrimSpace(latest.DeviceName)
	}
	if !latest.CollectedAt.IsZero() {
		collectData.Timestamp = latest.CollectedAt
	}
	if len(latest.Fields) > 0 {
		collectData.Fields = latest.Fields
	}

	return collectData
}

func buildSyncValues(fields map[string]string) map[string]any {
	fieldKeys := make([]string, 0, len(fields))
	for key := range fields {
		key = strings.TrimSpace(key)
		if key != "" {
			fieldKeys = append(fieldKeys, key)
		}
	}
	slices.Sort(fieldKeys)

	values := make(map[string]any, len(fieldKeys))
	for _, key := range fieldKeys {
		values[key] = convertFieldValue(fields[key])
	}
	return values
}

func syncTimestampOrNow(ts time.Time, nowMS int64) int64 {
	timestamp := ts.UnixMilli()
	if timestamp > 0 {
		return timestamp
	}
	return nowMS
}

func sortSyncSubDevices(subDevices []pandaXSyncSubDevice) {
	slices.SortFunc(subDevices, func(a, b pandaXSyncSubDevice) int {
		if c := cmp.Compare(strings.TrimSpace(a.ProductKey), strings.TrimSpace(b.ProductKey)); c != 0 {
			return c
		}
		return cmp.Compare(strings.TrimSpace(a.DeviceName), strings.TrimSpace(b.DeviceName))
	})
}

func resolveSyncSubDeviceProductKey(device *models.Device, productKeyByDeviceID map[int64]string) string {
	if device == nil {
		return ""
	}
	if resolved := strings.TrimSpace(productKeyByDeviceID[device.ID]); resolved != "" {
		return resolved
	}
	return strings.TrimSpace(device.ProductKey)
}

func resolveSyncProductKeyByDeviceID(devices []*models.Device) map[int64]string {
	result := make(map[int64]string, len(devices))
	if len(devices) == 0 || database.ParamDB == nil {
		return result
	}

	deviceIDsByDriver := make(map[int64][]int64)
	for _, dev := range devices {
		if dev == nil || dev.ID <= 0 || dev.DriverID == nil || *dev.DriverID <= 0 {
			continue
		}
		driverID := *dev.DriverID
		deviceIDsByDriver[driverID] = append(deviceIDsByDriver[driverID], dev.ID)
	}
	if len(deviceIDsByDriver) == 0 {
		return result
	}

	drivers, err := database.ListDrivers()
	if err != nil {
		slog.Info("PandaX resolve sync product key failed to load drivers", "error", err)
		return result
	}

	driverByID := make(map[int64]*models.Driver, len(drivers))
	for _, drv := range drivers {
		if drv == nil || drv.ID <= 0 {
			continue
		}
		driverByID[drv.ID] = drv
	}

	for driverID, deviceIDs := range deviceIDsByDriver {
		drv := driverByID[driverID]
		if drv == nil {
			continue
		}
		productKey := resolveDriverProductKey(drv)
		if productKey == "" {
			continue
		}
		for _, deviceID := range deviceIDs {
			if deviceID > 0 {
				result[deviceID] = productKey
			}
		}
	}

	return result
}

func resolveDriverProductKey(driver *models.Driver) string {
	if driver == nil {
		return ""
	}

	if productKey := strings.TrimSpace(driver.ProductKey); productKey != "" {
		return productKey
	}

	wasmPath := strings.TrimSpace(driver.FilePath)
	if wasmPath == "" {
		name := strings.TrimSpace(driver.Name)
		if name == "" {
			return ""
		}
		wasmPath = filepath.Join("drivers", name+".wasm")
	}

	wasmData, err := os.ReadFile(wasmPath)
	if err != nil {
		return ""
	}
	_, productKey, err := driverpkg.ExtractDriverMetadata(wasmData)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(productKey)
}

func validateSyncSubDevices(subDevices []pandaXSyncSubDevice) error {
	if len(subDevices) == 0 {
		return fmt.Errorf("同步预检查失败: 无可同步子设备")
	}

	type syncProductStats struct {
		total      int
		withFields int
	}

	statsByProduct := make(map[string]*syncProductStats, len(subDevices))
	for _, item := range subDevices {
		productKey := strings.TrimSpace(item.ProductKey)
		if productKey == "" {
			productKey = "unknown"
		}

		stats := statsByProduct[productKey]
		if stats == nil {
			stats = &syncProductStats{}
			statsByProduct[productKey] = stats
		}
		stats.total++
		if len(item.Values) > 0 {
			stats.withFields++
		}
	}

	emptyProducts := make([]string, 0)
	for productKey, stats := range statsByProduct {
		if stats == nil || stats.total == 0 {
			continue
		}
		if stats.withFields == 0 {
			emptyProducts = append(emptyProducts, productKey)
		}
	}
	if len(emptyProducts) > 0 {
		slices.Sort(emptyProducts)
		return fmt.Errorf("同步预检查失败: 产品[%s]无可用遥测字段，请先采集数据后再同步", strings.Join(emptyProducts, ","))
	}

	return nil
}
