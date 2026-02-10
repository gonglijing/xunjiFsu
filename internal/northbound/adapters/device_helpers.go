package adapters

import "strconv"

func defaultDeviceToken(deviceID int64) string {
	return "device_" + strconv.FormatInt(deviceID, 10)
}
