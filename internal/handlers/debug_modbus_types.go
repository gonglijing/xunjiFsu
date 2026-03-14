package handlers

const (
	modbusFuncReadHoldingRegisters = 0x03
	modbusFuncWriteSingleRegister  = 0x06
)

type modbusSerialDebugRequest struct {
	ResourceID    *int64 `json:"resource_id"`
	SerialPort    string `json:"serial_port"`
	BaudRate      int    `json:"baud_rate"`
	DataBits      int    `json:"data_bits"`
	StopBits      int    `json:"stop_bits"`
	Parity        string `json:"parity"`
	TimeoutMs     int    `json:"timeout_ms"`
	RawRequest    string `json:"raw_request"`
	ExpectRespLen int    `json:"expect_response_len"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       int    `json:"address"`
	Quantity      int    `json:"quantity"`
	Value         int    `json:"value"`
}

type modbusSerialDebugResponse struct {
	Port          string `json:"port"`
	RequestHex    string `json:"request_hex"`
	ResponseHex   string `json:"response_hex"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       *int   `json:"address,omitempty"`
	Quantity      *int   `json:"quantity,omitempty"`
	Value         *int   `json:"value,omitempty"`
	Registers     []int  `json:"registers,omitempty"`
	ExceptionCode *int   `json:"exception_code,omitempty"`
}

type modbusTCPDebugRequest struct {
	ResourceID    *int64 `json:"resource_id"`
	Endpoint      string `json:"endpoint"`
	TimeoutMs     int    `json:"timeout_ms"`
	RawRequest    string `json:"raw_request"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       int    `json:"address"`
	Quantity      int    `json:"quantity"`
	Value         int    `json:"value"`
	TransactionID int    `json:"transaction_id"`
}

type modbusTCPDebugResponse struct {
	Endpoint      string `json:"endpoint"`
	RequestHex    string `json:"request_hex"`
	ResponseHex   string `json:"response_hex"`
	TransactionID int    `json:"transaction_id"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       *int   `json:"address,omitempty"`
	Quantity      *int   `json:"quantity,omitempty"`
	Value         *int   `json:"value,omitempty"`
	Registers     []int  `json:"registers,omitempty"`
	ExceptionCode *int   `json:"exception_code,omitempty"`
}
