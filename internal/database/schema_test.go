package database

import "testing"

func TestParseIndexTableName(t *testing.T) {
	tests := []struct {
		name string
		stmt string
		want string
	}{
		{
			name: "plain table",
			stmt: "CREATE INDEX idx_device_id ON data_points(device_id);",
			want: "data_points",
		},
		{
			name: "quoted schema table",
			stmt: "CREATE INDEX idx_alarm ON `main`.`alarm_logs` (`device_id`);",
			want: "alarm_logs",
		},
		{
			name: "missing on clause",
			stmt: "CREATE INDEX idx_invalid device_id;",
			want: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := parseIndexTableName(testCase.stmt); got != testCase.want {
				t.Fatalf("parseIndexTableName() = %q, want %q", got, testCase.want)
			}
		})
	}
}
