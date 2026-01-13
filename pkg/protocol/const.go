package protocol

// Protocol constants defining the message format.
const (
	// ProtocolVersion is the current protocol version.
	ProtocolVersion uint8 = 1
	// HeaderSize is the size of the message header in bytes.
	HeaderSize int = 18
	// MaxPayloadSize is the maximum allowed payload size (16 MB).
	MaxPayloadSize uint32 = 16 * 1024 * 1024
)

// MagicBytes is the protocol identifier "WEBS".
var MagicBytes = [4]byte{0x57, 0x45, 0x42, 0x53}
