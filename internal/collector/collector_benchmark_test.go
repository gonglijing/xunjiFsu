package collector

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func makeBenchmarkDriverResult(pointCount int) *driver.DriverResult {
	points := make([]driver.DriverPoint, 0, pointCount)
	for i := 0; i < pointCount; i++ {
		points = append(points, driver.DriverPoint{
			FieldName: fmt.Sprintf("p_%d", i),
			Value:     float64(i) + 0.5,
		})
	}
	return &driver.DriverResult{
		ProductKey: "bench-pk",
		Points:     points,
	}
}

func benchmarkDriverResultToCollectData(b *testing.B, pointCount int) {
	device := &models.Device{
		ID:         1,
		Name:       "bench-device",
		ProductKey: "device-pk",
		DeviceKey:  "device-dk",
	}
	result := makeBenchmarkDriverResult(pointCount)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collect := driverResultToCollectData(device, result)
		if collect == nil || len(collect.Points) != pointCount {
			b.Fatalf("unexpected collect result size: %d", len(collect.Points))
		}
	}
}

func benchmarkDriverResultToCollectDataMixed(b *testing.B, pointCount int) {
	device := &models.Device{
		ID:         1,
		Name:       "bench-device",
		ProductKey: "device-pk",
		DeviceKey:  "device-dk",
	}
	result := makeBenchmarkDriverResult(pointCount)
	result.Data = make(map[string]string, pointCount)
	for i := 0; i < pointCount; i++ {
		result.Data[fmt.Sprintf("d_%d", i)] = fmt.Sprintf("%d", i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collect := driverResultToCollectData(device, result)
		fields := collect.EnsureFields()
		if collect == nil || len(collect.Points) != pointCount || len(fields) != pointCount*2 {
			b.Fatalf("unexpected collect result sizes: points=%d fields=%d", len(collect.Points), len(fields))
		}
	}
}

func benchmarkDriverResultToCollectDataMixedDeferred(b *testing.B, pointCount int) {
	device := &models.Device{
		ID:         1,
		Name:       "bench-device",
		ProductKey: "device-pk",
		DeviceKey:  "device-dk",
	}
	result := makeBenchmarkDriverResult(pointCount)
	result.Data = make(map[string]string, pointCount)
	for i := 0; i < pointCount; i++ {
		result.Data[fmt.Sprintf("d_%d", i)] = fmt.Sprintf("%d", i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collect := driverResultToCollectData(device, result)
		if collect == nil || len(collect.Points) != pointCount || len(collect.Fields) != pointCount {
			b.Fatalf("unexpected collect result sizes: points=%d fields=%d", len(collect.Points), len(collect.Fields))
		}
	}
}

func BenchmarkDriverResultToCollectData_1000Points(b *testing.B) {
	benchmarkDriverResultToCollectData(b, 1000)
}

func BenchmarkDriverResultToCollectData_10000Points(b *testing.B) {
	benchmarkDriverResultToCollectData(b, 10000)
}

func BenchmarkDriverResultToCollectDataMixed_1000Points(b *testing.B) {
	benchmarkDriverResultToCollectDataMixed(b, 1000)
}

func BenchmarkDriverResultToCollectDataMixed_10000Points(b *testing.B) {
	benchmarkDriverResultToCollectDataMixed(b, 10000)
}

func BenchmarkDriverResultToCollectDataMixedDeferred_1000Points(b *testing.B) {
	benchmarkDriverResultToCollectDataMixedDeferred(b, 1000)
}

func BenchmarkDriverResultToCollectDataMixedDeferred_10000Points(b *testing.B) {
	benchmarkDriverResultToCollectDataMixedDeferred(b, 10000)
}

func BenchmarkCollectDataEnsureFieldsRepeated_10000Points(b *testing.B) {
	device := &models.Device{
		ID:         1,
		Name:       "bench-device",
		ProductKey: "device-pk",
		DeviceKey:  "device-dk",
	}
	result := makeBenchmarkDriverResult(10000)
	result.Data = make(map[string]string, 10000)
	for i := 0; i < 10000; i++ {
		result.Data[fmt.Sprintf("d_%d", i)] = fmt.Sprintf("%d", i)
	}

	collect := driverResultToCollectData(device, result)
	fields := collect.EnsureFields()
	if len(fields) != 20000 {
		b.Fatalf("unexpected fields size: %d", len(fields))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := collect.EnsureFields(); len(got) != 20000 {
			b.Fatalf("unexpected fields size on repeat: %d", len(got))
		}
	}
}

func BenchmarkNumericFieldLookupPointCached_100Rules(b *testing.B) {
	points := make([]models.CollectPoint, 0, 100)
	fields := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("p_%d", i)
		points = append(points, models.CollectPoint{
			FieldName: " " + name + " ",
			Value:     float64(i) + 0.5,
		})
		fields = append(fields, name)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lookup := newNumericFieldLookup(nil, points)
		for _, field := range fields {
			if value, ok := lookup.getFloatWithNormalized(field, field); !ok || value < 0 {
				b.Fatalf("unexpected lookup result for %s: (%v, %v)", field, value, ok)
			}
		}
	}
}

func BenchmarkCheckThresholds_100RulesNoMatch(b *testing.B) {
	resetThresholdCache()
	deviceID := int64(9001)
	device := &models.Device{ID: deviceID, Name: "bench-threshold-device"}
	data := &models.CollectData{
		DeviceID: deviceID,
		Points:   make([]models.CollectPoint, 0, 100),
	}
	rules := make([]thresholdEvalRule, 0, 100)
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("p_%d", i)
		threshold := &models.Threshold{
			ID:        int64(i + 1),
			DeviceID:  deviceID,
			FieldName: name,
			Operator:  ">",
			Value:     1000,
		}
		rules = append(rules, buildThresholdEvalRule(threshold))
		data.Points = append(data.Points, models.CollectPoint{
			FieldName: name,
			Value:     float64(i),
		})
	}

	cache.mu.Lock()
	cache.rules[deviceID] = rules
	cache.thresholds[deviceID] = make([]*models.Threshold, 0, len(rules))
	cache.lastRefresh = time.Now()
	cache.mu.Unlock()
	clearAlarmStateForDevice(deviceID)
	defer func() {
		clearAlarmStateForDevice(deviceID)
		resetThresholdCache()
	}()

	collector := NewCollector(nil, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := collector.checkThresholds(device, data); err != nil {
			b.Fatalf("checkThresholds failed: %v", err)
		}
	}
}

func BenchmarkCheckThresholds_100RulesMatchedSuppressed(b *testing.B) {
	resetThresholdCache()
	deviceID := int64(9002)
	device := &models.Device{ID: deviceID, Name: "bench-threshold-device"}
	data := &models.CollectData{
		DeviceID: deviceID,
		Points:   make([]models.CollectPoint, 0, 100),
	}
	rules := make([]thresholdEvalRule, 0, 100)
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("p_%d", i)
		threshold := &models.Threshold{
			ID:        int64(i + 1),
			DeviceID:  deviceID,
			FieldName: name,
			Operator:  ">",
			Value:     -1,
		}
		rules = append(rules, buildThresholdEvalRule(threshold))
		data.Points = append(data.Points, models.CollectPoint{
			FieldName: name,
			Value:     float64(i),
		})
	}

	cache.mu.Lock()
	cache.rules[deviceID] = rules
	cache.thresholds[deviceID] = make([]*models.Threshold, 0, len(rules))
	cache.lastRefresh = time.Now()
	cache.mu.Unlock()
	clearAlarmStateForDevice(deviceID)
	defer func() {
		clearAlarmStateForDevice(deviceID)
		resetThresholdCache()
	}()

	now := time.Now()
	alarmStates.mu.Lock()
	for _, rule := range rules {
		key := rule.alarmKey
		key.DeviceID = deviceID
		alarmStates.data[key] = alarmState{LastTriggered: now}
	}
	alarmStates.mu.Unlock()

	collector := NewCollector(nil, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := collector.checkThresholds(device, data); err != nil {
			b.Fatalf("checkThresholds failed: %v", err)
		}
	}
}
