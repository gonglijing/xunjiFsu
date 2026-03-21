package app

import "net/http"

func registerDebugRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("POST /debug/modbus/serial", apiDeps.debugModbus.DebugModbusSerial)
	api.HandleFunc("POST /debug/modbus/tcp", apiDeps.debugModbus.DebugModbusTCP)
}
