package handlers

const (
	errInvalidRequestBodyWithDetailPrefix = "Invalid request body: "
	errInvalidDeviceIDMessage             = "Invalid device_id"

	errDeviceNameRequiredMessage = "device name is required"
	errDriverNameRequiredMessage = "driver name is required"

	errDeviceNotFoundMessage           = "Device not found"
	errDriverNotFoundMessage           = "Driver not found"
	errNorthboundConfigNotFoundMessage = "Northbound config not found"
	errResourceNotFoundMessage         = "resource not found"

	errDriverWasmFileNotFoundMessage      = "driver wasm file not found"
	errOnlyWasmFilesAllowedMessage        = "Only .wasm files are allowed"
	errDriverLookupFailedMessage          = "driver not found"
	errDriverNotLoadedMessage             = "driver is not loaded"
	errDriverConfigSchemaInvalidJSONError = "driver config_schema is invalid JSON"
	errDeviceHasNoDriverMessage           = "Device has no driver"

	errGatewayIdentityRequiredMessage = "请先在网关配置中设置 product_key 和 device_key"
)

var (
	apiErrInvalidID = APIErrorDef{
		Code:    "E_INVALID_ID",
		Message: "Invalid ID",
	}
	apiErrInvalidRequestBody = APIErrorDef{
		Code:    "E_INVALID_REQUEST_BODY",
		Message: "Invalid request body",
	}
	apiErrInvalidDeviceID = APIErrorDef{
		Code:    "E_INVALID_DEVICE_ID",
		Message: errInvalidDeviceIDMessage,
	}
	apiErrDeviceNotFound = APIErrorDef{
		Code:    "E_DEVICE_NOT_FOUND",
		Message: errDeviceNotFoundMessage,
	}
	apiErrDriverNotFound = APIErrorDef{
		Code:    "E_DRIVER_NOT_FOUND",
		Message: errDriverNotFoundMessage,
	}
	apiErrNorthboundConfigNotFound = APIErrorDef{
		Code:    "E_NORTHBOUND_NOT_FOUND",
		Message: errNorthboundConfigNotFoundMessage,
	}
	apiErrResourceNotFound = APIErrorDef{
		Code:    "E_RESOURCE_NOT_FOUND",
		Message: errResourceNotFoundMessage,
	}
)
