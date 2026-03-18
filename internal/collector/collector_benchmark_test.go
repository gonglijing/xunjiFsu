package collector

import (
	"fmt"
	"testing"

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
