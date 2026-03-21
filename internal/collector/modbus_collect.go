package collector

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (c *Collector) canCollectTask(task *collectTask) bool {
	return task != nil && task.device != nil && c.driverExecutor != nil
}

func (c *Collector) collectDataFromDriver(task *collectTask) (*models.CollectData, error) {
	if task == nil || task.device == nil {
		return nil, fmt.Errorf("collect task is nil")
	}

	device := task.device
	result, err := c.driverExecutor.ExecutePrepared(device, "handle", task.preparedRead, nil)
	if err != nil {
		return nil, err
	}

	collect := driverResultToCollectDataWithTask(task, result)
	if err := c.syncDeviceProductKey(device, collect); err != nil {
		slog.Warn("Failed to sync device product_key from driver output", "error", err)
	}
	return collect, nil
}

func (c *Collector) persistCollectData(task *collectTask, collect *models.CollectData) {
	if task == nil || collect == nil {
		return
	}

	storeHistory := shouldStoreHistory(task, collect.Timestamp)
	// 实时缓存始终写入，历史按设备 storage_interval 控制。
	if err := database.EnqueueCollectDataWrite(collect, storeHistory); err != nil {
		slog.Error("Failed to insert data points", "error", err)
	}
	c.markTaskCollected(task, collect.Timestamp, storeHistory)
}

// handleThresholdForDevice 仅检查阈值（用于采集时触发报警）
func (c *Collector) handleThresholdForDevice(device *models.Device, data *models.CollectData) {
	if device == nil || data == nil {
		return
	}
	if err := c.checkThresholds(device, data); err != nil {
		slog.Error("check thresholds error", "error", err)
	}
}

// driverResultToCollectData 转换驱动结果为采集数据
func driverResultToCollectData(device *models.Device, res *driver.DriverResult) *models.CollectData {
	return driverResultToCollectDataWithCache(device, res, trimCollectorText(device.ProductKey), trimCollectorText(device.DeviceKey))
}

func driverResultToCollectDataWithTask(task *collectTask, res *driver.DriverResult) *models.CollectData {
	if task == nil {
		return driverResultToCollectData(nil, res)
	}
	return driverResultToCollectDataWithCache(task.device, res, task.deviceProductKey, task.deviceKey)
}

func driverResultToCollectDataWithCache(device *models.Device, res *driver.DriverResult, deviceProductKey, deviceKey string) *models.CollectData {
	if res == nil {
		res = &driver.DriverResult{}
	}
	if device == nil {
		device = &models.Device{}
	}

	fields, points := resultFieldsForCollect(res)
	ts := res.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	resultProductKey := trimCollectorText(res.ProductKey)
	if resultProductKey == "" {
		resultProductKey = deviceProductKey
	}
	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		ProductKey: resultProductKey,
		DeviceKey:  deviceKey,
		Timestamp:  ts,
		Fields:     fields,
		Points:     points,
	}
}

func (c *Collector) syncDeviceProductKey(device *models.Device, collect *models.CollectData) error {
	if device == nil || collect == nil {
		return nil
	}
	nextProductKey := c.resolveFixedDriverProductKey(device.DriverID, collect.ProductKey)
	if nextProductKey == "" {
		return nil
	}
	collect.ProductKey = nextProductKey
	currentProductKey := trimCollectorText(device.ProductKey)
	if currentProductKey == nextProductKey {
		return nil
	}
	if database.ParamDB != nil {
		if err := database.UpdateDeviceProductKey(device.ID, nextProductKey); err != nil {
			return err
		}
	}
	device.ProductKey = nextProductKey
	task := c.findTask(device.ID)
	if task != nil {
		task.deviceProductKey = nextProductKey
	}
	return nil
}

func (c *Collector) resolveFixedDriverProductKey(driverID *int64, candidate string) string {
	candidate = trimCollectorText(candidate)
	if driverID == nil || *driverID <= 0 {
		return candidate
	}

	id := *driverID
	c.driverIdentityMu.Lock()
	defer c.driverIdentityMu.Unlock()

	cached := trimCollectorText(c.driverProductKeys[id])
	if cached == "" {
		if candidate != "" {
			c.driverProductKeys[id] = candidate
		}
		return candidate
	}

	if candidate != "" && candidate != cached {
		slog.Warn("Collector: driver productKey mismatch, use cached", "driver_id", id, "cached", cached, "incoming", candidate)
	}
	return cached
}

func resultFieldsForCollect(res *driver.DriverResult) (map[string]string, []models.CollectPoint) {
	if res == nil {
		return nil, nil
	}

	if len(res.Points) == 0 {
		return normalizedDataFields(res.Data), nil
	}
	points := normalizedCollectPoints(res.Points)
	if len(res.Data) == 0 {
		return nil, points
	}

	return normalizedDataFields(res.Data), points
}

func normalizedDataFields(data map[string]string) map[string]string {
	return normalizedDataFieldsWithExtraCap(data, 0)
}

func normalizedDataFieldsWithExtraCap(data map[string]string, extraCap int) map[string]string {
	if len(data) == 0 {
		return nil
	}
	if canReuseCollectedDataFields(data) {
		return data
	}

	var fields map[string]string
	for key, value := range data {
		name := trimCollectorText(key)
		if name == "" {
			continue
		}
		if fields == nil {
			fields = make(map[string]string, len(data)+extraCap)
		}
		fields[name] = value
	}
	return fields
}

func normalizedCollectPoints(points []driver.DriverPoint) []models.CollectPoint {
	if len(points) == 0 {
		return nil
	}
	if canReuseCollectedPoints(points) {
		return points
	}

	write := 0
	for i := range points {
		name := trimCollectorText(points[i].FieldName)
		if name == "" {
			continue
		}
		points[i].FieldName = name
		points[write] = points[i]
		write++
	}
	if write == 0 {
		return nil
	}
	return points[:write]
}

func canReuseCollectedPoints(points []driver.DriverPoint) bool {
	for i := range points {
		name := trimCollectorText(points[i].FieldName)
		if name == "" {
			return false
		}
		if points[i].FieldName != name {
			return false
		}
	}
	return true
}

func canReuseCollectedDataFields(data map[string]string) bool {
	for key := range data {
		trimmed := trimCollectorText(key)
		if trimmed == "" {
			return false
		}
		if key != trimmed {
			return false
		}
	}
	return true
}

func trimCollectorText(s string) string {
	if s == "" {
		return ""
	}
	start := 0
	end := len(s)

	for start < end {
		c := s[start]
		if !isASCIICollectorSpace(c) {
			break
		}
		start++
	}
	if start == end {
		return ""
	}
	for end > start {
		c := s[end-1]
		if !isASCIICollectorSpace(c) {
			break
		}
		end--
	}
	if start == 0 && end == len(s) {
		if s[0] < utf8.RuneSelf && s[len(s)-1] < utf8.RuneSelf {
			return s
		}
		return strings.TrimSpace(s)
	}
	if s[start] < utf8.RuneSelf && s[end-1] < utf8.RuneSelf {
		return s[start:end]
	}
	return strings.TrimSpace(s)
}

func isASCIICollectorSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}
