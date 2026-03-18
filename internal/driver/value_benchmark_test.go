package driver

import "testing"

func BenchmarkResultFields_ModbusPoints_256(b *testing.B) {
	points := make([]DriverPoint, 0, 258)
	for i := 0; i < 256; i++ {
		points = append(points, DriverPoint{
			FieldName: "reg_" + formatDriverValue(i),
			Value:     i,
		})
	}
	points = append(points,
		DriverPoint{FieldName: " productKey ", Value: "pk"},
		DriverPoint{FieldName: "   ", Value: "x"},
	)
	result := &DriverResult{Points: points}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fields := ResultFields(result)
		if len(fields) != 256 {
			b.Fatalf("unexpected fields len: %d", len(fields))
		}
	}
}

func BenchmarkResultFields_DirtyData_256(b *testing.B) {
	data := make(map[string]string, 258)
	for i := 0; i < 256; i++ {
		data[" reg_"+formatDriverValue(i)+" "] = formatDriverValue(i)
	}
	data[" product_key "] = "pk"
	data["   "] = "x"
	result := &DriverResult{Data: data}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fields := ResultFields(result)
		if len(fields) != 256 {
			b.Fatalf("unexpected fields len: %d", len(fields))
		}
	}
}
