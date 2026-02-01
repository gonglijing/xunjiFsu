package collector

import (
	"sync"
)

// DataPointPool 数据点对象池
// 减少GC压力，提高性能
var DataPointPool = sync.Pool{
	New: func() interface{} {
		return &pooledDataPoint{}
	},
}

// pooledDataPoint 可复用的数据点
type pooledDataPoint struct {
	DeviceID    int64
	DeviceName  string
	FieldName   string
	Value       string
	ValueType   string
	CollectedAt int64 // 使用UnixNano减少内存占用
}

// GetDataPoint 从池中获取数据点
func GetDataPoint() *pooledDataPoint {
	return DataPointPool.Get().(*pooledDataPoint)
}

// PutDataPoint 将数据点归还到池中
func PutDataPoint(p *pooledDataPoint) {
	if p == nil {
		return
	}
	// 重置字段
	p.DeviceID = 0
	p.DeviceName = ""
	p.FieldName = ""
	p.Value = ""
	p.ValueType = ""
	p.CollectedAt = 0
	DataPointPool.Put(p)
}

// DataPointBuffer 数据点缓冲区（批量处理）
type DataPointBuffer struct {
	mu       sync.Mutex
	points   []*pooledDataPoint
	capacity int
}

// NewDataPointBuffer 创建缓冲区
func NewDataPointBuffer(capacity int) *DataPointBuffer {
	return &DataPointBuffer{
		points:   make([]*pooledDataPoint, 0, capacity),
		capacity: capacity,
	}
}

// Add 添加数据点
func (b *DataPointBuffer) Add(deviceID int64, deviceName, fieldName, value, valueType string, collectedAt int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	p := GetDataPoint()
	p.DeviceID = deviceID
	p.DeviceName = deviceName
	p.FieldName = fieldName
	p.Value = value
	p.ValueType = valueType
	p.CollectedAt = collectedAt

	b.points = append(b.points, p)
}

// Flush 刷新缓冲区并返回所有数据点
func (b *DataPointBuffer) Flush() []*pooledDataPoint {
	b.mu.Lock()
	defer b.mu.Unlock()

	points := b.points
	b.points = make([]*pooledDataPoint, 0, b.capacity)
	return points
}

// Len 返回缓冲区中的数据点数量
func (b *DataPointBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.points)
}

// IsFull 检查缓冲区是否已满
func (b *DataPointBuffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.points) >= b.capacity
}

// TaskPool 任务对象池
var TaskPool = sync.Pool{
	New: func() interface{} {
		return &pooledTask{}
	},
}

type pooledTask struct {
	deviceID   int64
	deviceName string
	interval   int64 // 毫秒
	lastRun    int64
	lastUpload int64
}

// GetTask 从池中获取任务
func GetTask() *pooledTask {
	return TaskPool.Get().(*pooledTask)
}

// PutTask 将任务归还到池中
func PutTask(t *pooledTask) {
	if t == nil {
		return
	}
	t.deviceID = 0
	t.deviceName = ""
	t.interval = 0
	t.lastRun = 0
	t.lastUpload = 0
	TaskPool.Put(t)
}
