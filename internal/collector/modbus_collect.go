package collector

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (c *Collector) canCollectTask(task *collectTask) bool {
	return task != nil && task.device != nil && c.driverExecutor != nil
}

func (c *Collector) collectDataFromDriver(device *models.Device) (*models.CollectData, error) {
	result, err := c.driverExecutor.Execute(device)
	if err != nil {
		return nil, err
	}

	collect := driverResultToCollectData(device, result)
	if err := c.syncDeviceProductKey(device, collect); err != nil {
		log.Printf("Failed to sync device product_key from driver output: %v", err)
	}
	return collect, nil
}

func (c *Collector) persistCollectData(task *collectTask, collect *models.CollectData) {
	if task == nil || collect == nil {
		return
	}

	storeHistory := shouldStoreHistory(task, collect.Timestamp)
	// 实时缓存始终写入，历史按设备 storage_interval 控制。
	if err := database.InsertCollectDataWithOptions(collect, storeHistory); err != nil {
		log.Printf("Failed to insert data points: %v", err)
	}
	c.markTaskCollected(task, collect.Timestamp, storeHistory)
}

// handleThresholdForDevice 仅检查阈值（用于采集时触发报警）
func (c *Collector) handleThresholdForDevice(device *models.Device, data *models.CollectData) {
	if device == nil || data == nil {
		return
	}
	if err := c.checkThresholds(device, data); err != nil {
		log.Printf("check thresholds error: %v", err)
	}
}

// driverResultToCollectData 转换驱动结果为采集数据
func driverResultToCollectData(device *models.Device, res *driver.DriverResult) *models.CollectData {
	if res == nil {
		res = &driver.DriverResult{}
	}

	fields := resultFieldsForCollect(res)
	ts := res.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	resultProductKey := strings.TrimSpace(res.ProductKey)
	deviceProductKey := strings.TrimSpace(device.ProductKey)
	if resultProductKey == "" {
		resultProductKey = deviceProductKey
	}
	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		ProductKey: resultProductKey,
		DeviceKey:  strings.TrimSpace(device.DeviceKey),
		Timestamp:  ts,
		Fields:     fields,
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
	currentProductKey := strings.TrimSpace(device.ProductKey)
	if currentProductKey == nextProductKey {
		return nil
	}
	if database.ParamDB != nil {
		if err := database.UpdateDeviceProductKey(device.ID, nextProductKey); err != nil {
			return err
		}
	}
	device.ProductKey = nextProductKey
	return nil
}

func (c *Collector) resolveFixedDriverProductKey(driverID *int64, candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if driverID == nil || *driverID <= 0 {
		return candidate
	}

	id := *driverID
	c.driverIdentityMu.Lock()
	defer c.driverIdentityMu.Unlock()

	cached := strings.TrimSpace(c.driverProductKeys[id])
	if cached == "" {
		if candidate != "" {
			c.driverProductKeys[id] = candidate
		}
		return candidate
	}

	if candidate != "" && candidate != cached {
		log.Printf("Collector: driver %d productKey mismatch (cached=%s incoming=%s), use cached", id, cached, candidate)
	}
	return cached
}

func resultFieldsForCollect(res *driver.DriverResult) map[string]string {
	if res == nil {
		return nil
	}

	if len(res.Points) == 0 {
		return normalizedDataFields(res.Data)
	}

	fields := normalizedDataFields(res.Data)
	if fields == nil {
		fields = make(map[string]string, len(res.Points))
	}
	for _, p := range res.Points {
		name := strings.TrimSpace(p.FieldName)
		if name == "" {
			continue
		}
		fields[name] = driverPointValueToString(p.Value)
	}
	return fields
}

func normalizedDataFields(data map[string]string) map[string]string {
	if len(data) == 0 {
		return nil
	}
	if canReuseCollectedDataFields(data) {
		return data
	}

	var fields map[string]string
	for key, value := range data {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		if fields == nil {
			fields = make(map[string]string, len(data))
		}
		fields[name] = value
	}
	return fields
}

func canReuseCollectedDataFields(data map[string]string) bool {
	for key := range data {
		if strings.TrimSpace(key) == "" {
			return false
		}
		if key != strings.TrimSpace(key) {
			return false
		}
	}
	return true
}

func driverPointValueToString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
