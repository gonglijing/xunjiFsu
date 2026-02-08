package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type historyDataQuery struct {
	DeviceID  *int64
	FieldName string
	StartTime time.Time
	EndTime   time.Time
}

func parseHistoryDataQuery(r *http.Request) (historyDataQuery, error) {
	query := historyDataQuery{
		FieldName: strings.TrimSpace(r.URL.Query().Get("field_name")),
	}

	deviceID, err := parseOptionalInt64Query(r, "device_id")
	if err != nil {
		return historyDataQuery{}, fmt.Errorf(errInvalidDeviceIDMessage)
	}
	query.DeviceID = deviceID

	startTime, err := parseOptionalTimeQuery(r, "start")
	if err != nil {
		return historyDataQuery{}, fmt.Errorf("Invalid start time")
	}
	query.StartTime = startTime

	endTime, err := parseOptionalTimeQuery(r, "end")
	if err != nil {
		return historyDataQuery{}, fmt.Errorf("Invalid end time")
	}
	query.EndTime = endTime

	if err := validateHistoryDataQuery(query); err != nil {
		return historyDataQuery{}, err
	}

	return query, nil
}

func validateHistoryDataQuery(query historyDataQuery) error {
	if query.DeviceID != nil && *query.DeviceID <= 0 {
		return fmt.Errorf(errInvalidDeviceIDMessage)
	}

	if !query.StartTime.IsZero() && !query.EndTime.IsZero() && query.StartTime.After(query.EndTime) {
		return fmt.Errorf(errHistoryStartAfterEndDetail)
	}

	if query.DeviceID == nil {
		hasFilter := query.FieldName != "" || !query.StartTime.IsZero() || !query.EndTime.IsZero()
		if hasFilter {
			return fmt.Errorf(errHistoryFilterRequiresDevice)
		}
	}

	return nil
}

func parseOptionalInt64Query(r *http.Request, key string) (*int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseOptionalTimeQuery(r *http.Request, key string) (time.Time, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	return parseTimeParam(raw)
}

func parseTimeParam(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02T15:04", value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("invalid time format")
}
