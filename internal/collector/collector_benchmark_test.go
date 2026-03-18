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

func BenchmarkDriverResultToCollectData_1000Points(b *testing.B) {
	benchmarkDriverResultToCollectData(b, 1000)
}

func BenchmarkDriverResultToCollectData_10000Points(b *testing.B) {
	benchmarkDriverResultToCollectData(b, 10000)
}
