package app

import "net/http"

func registerAlarmRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /alarms", apiDeps.alarm.GetAlarmLogs)
	api.HandleFunc("DELETE /alarms", apiDeps.alarm.ClearAlarms)
	api.HandleFunc("POST /alarms/batch-delete", apiDeps.alarm.BatchDeleteAlarms)
	api.HandleFunc("DELETE /alarms/{id}", apiDeps.alarm.DeleteAlarm)
	api.HandleFunc("POST /alarms/{id}/acknowledge", apiDeps.alarm.AcknowledgeAlarm)
}
