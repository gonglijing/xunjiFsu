package httpapi

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

func writeModbusDebugParamError(w http.ResponseWriter, err error) {
	WriteBadRequestCode(w, errDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", errDebugModbusParamInvalid.Message, err))
}

func writeModbusDebugResolveError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		WriteNotFoundDef(w, apiErrResourceNotFound)
		return
	}
	writeModbusDebugParamError(w, err)
}

func writeModbusDebugCommError(w http.ResponseWriter, def APIErrorDef, err error) {
	WriteErrorCode(w, http.StatusBadGateway, def.Code, fmt.Sprintf("%s: %v", def.Message, err))
}

func writeModbusDebugResponseError(w http.ResponseWriter, err error) {
	WriteErrorCode(w, http.StatusBadGateway, errDebugModbusResponseInvalid.Code, fmt.Sprintf("%s: %v", errDebugModbusResponseInvalid.Message, err))
}
