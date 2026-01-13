package protocol

// Opcode represents message type identifiers in the WebOS protocol.
// Each opcode corresponds to a specific type of message or operation.
type Opcode uint8

// Protocol opcodes defining message types.
const (
	// OpcodeInvalid represents an invalid or unspecified opcode.
	OpcodeInvalid Opcode = iota
	// OpcodeDisplay is used for display rendering instructions (server → client).
	OpcodeDisplay
	// OpcodeInput is used for keyboard/mouse input events (client → server).
	OpcodeInput
	// OpcodeFileSystem is used for file system operations.
	OpcodeFileSystem
	// OpcodeNetwork is used for network operations.
	OpcodeNetwork
	// OpcodeProcess is used for process management.
	OpcodeProcess
	// OpcodeAuth is used for authentication messages.
	OpcodeAuth
	// OpcodeConnect is used for connection establishment.
	OpcodeConnect
	// OpcodeDisconnect is used for connection termination.
	OpcodeDisconnect
	// OpcodePing is used for keep-alive ping messages.
	OpcodePing
	// OpcodePong is used for keep-alive pong responses.
	OpcodePong
	// OpcodeError is used for error messages.
	OpcodeError
)

// String returns the string representation of the opcode.
func (o Opcode) String() string {
	switch o {
	case OpcodeDisplay:
		return "DISPLAY"
	case OpcodeInput:
		return "INPUT"
	case OpcodeFileSystem:
		return "FILESYSTEM"
	case OpcodeNetwork:
		return "NETWORK"
	case OpcodeProcess:
		return "PROCESS"
	case OpcodeAuth:
		return "AUTH"
	case OpcodeConnect:
		return "CONNECT"
	case OpcodeDisconnect:
		return "DISCONNECT"
	case OpcodePing:
		return "PING"
	case OpcodePong:
		return "PONG"
	case OpcodeError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// IsValid checks if the opcode is a valid protocol opcode.
func (o Opcode) IsValid() bool {
	return o >= OpcodeDisplay && o <= OpcodeError
}
