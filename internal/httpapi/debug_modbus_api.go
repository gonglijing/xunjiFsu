package httpapi

type DebugModbusAPI struct{}

func NewDebugModbusAPI() *DebugModbusAPI {
	return &DebugModbusAPI{}
}

var (
	errDebugModbusParamInvalid    = APIErrorDef{Code: "E_DEBUG_MODBUS_PARAM_INVALID", Message: "串口 Modbus 调试参数无效"}
	errDebugModbusSerialFailed    = APIErrorDef{Code: "E_DEBUG_MODBUS_SERIAL_FAILED", Message: "串口 Modbus 调试通信失败"}
	errDebugModbusTCPFailed       = APIErrorDef{Code: "E_DEBUG_MODBUS_TCP_FAILED", Message: "Modbus TCP 调试通信失败"}
	errDebugModbusResponseInvalid = APIErrorDef{Code: "E_DEBUG_MODBUS_RESPONSE_INVALID", Message: "Modbus 调试响应无效"}
)
