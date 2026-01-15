package dns

import (
	"net"
	"time"
)

// DNS header flags and opcodes
const (
	FlagQR    uint16 = 0x8000 // Query/Response flag
	FlagAA    uint16 = 0x0400 // Authoritative Answer
	FlagTC    uint16 = 0x0200 // Truncation Flag
	FlagRD    uint16 = 0x0100 // Recursion Desired
	FlagRA    uint16 = 0x0080 // Recursion Available
	FlagZ     uint16 = 0x0070 // Reserved Z flags
	FlagRCODE uint16 = 0x000F // Response Code
)

// DNS port for standard queries
const DefaultDNSPort = 53

// Default timeout for DNS queries
const DefaultTimeout = 5 * time.Second

// Default cache TTL
const DefaultCacheTTL = 5 * time.Minute

// Opcode represents a DNS opcode
type Opcode uint8

// DNS opcodes
const (
	OpcodeQuery  Opcode = 0 // Standard query
	OpcodeIQuery Opcode = 1 // Inverse query (deprecated)
	OpcodeStatus Opcode = 2 // Server status request
	OpcodeNotify Opcode = 4 // Zone change notification
	OpcodeUpdate Opcode = 5 // Dynamic update
)

// RCode represents a DNS response code
type RCode uint8

// DNS response codes
const (
	RCodeSuccess        RCode = 0  // No error
	RCodeFormatError    RCode = 1  // Format error
	RCodeServerFailure  RCode = 2  // Server failure
	RCodeNameError      RCode = 3  // Non-existent domain (NXDOMAIN)
	RCodeNotImplemented RCode = 4  // Not implemented
	RCodeRefused        RCode = 5  // Query refused
	RCodeNameExists     RCode = 6  // Name exists (for AXFR)
	RCodeRRSetExists    RCode = 7  // RR set exists
	RCodeRRSetNotExists RCode = 8  // RR set does not exist
	RCodeNotAuth        RCode = 9  // Not authoritative
	RCodeNotZone        RCode = 10 // Not a zone
)

// RecordType represents a DNS resource record type
type RecordType uint16

// DNS record types
const (
	RecordTypeA     RecordType = 1   // IPv4 address
	RecordTypeNS    RecordType = 2   // Name server
	RecordTypeCNAME RecordType = 5   // Canonical name
	RecordTypeSOA   RecordType = 6   // Start of authority
	RecordTypePTR   RecordType = 12  // Domain pointer
	RecordTypeHINFO RecordType = 13  // Host info
	RecordTypeMX    RecordType = 15  // Mail exchange
	RecordTypeTXT   RecordType = 16  // Text record
	RecordTypeAAAA  RecordType = 28  // IPv6 address
	RecordTypeSRV   RecordType = 33  // Service location
	RecordTypeANY   RecordType = 255 // Any record
)

// RecordClass represents a DNS record class
type RecordClass uint16

// DNS record classes
const (
	ClassIN  RecordClass = 1   // Internet class
	ClassCS  RecordClass = 2   // CSNET class (deprecated)
	ClassCH  RecordClass = 3   // CHAOS class
	ClassHS  RecordClass = 4   // Hesiod class
	ClassANY RecordClass = 255 // Any class
)

// String returns a string representation of the record type
func (rt RecordType) String() string {
	switch rt {
	case RecordTypeA:
		return "A"
	case RecordTypeNS:
		return "NS"
	case RecordTypeCNAME:
		return "CNAME"
	case RecordTypeSOA:
		return "SOA"
	case RecordTypePTR:
		return "PTR"
	case RecordTypeHINFO:
		return "HINFO"
	case RecordTypeMX:
		return "MX"
	case RecordTypeTXT:
		return "TXT"
	case RecordTypeAAAA:
		return "AAAA"
	case RecordTypeSRV:
		return "SRV"
	case RecordTypeANY:
		return "ANY"
	default:
		return "UNKNOWN"
	}
}

// String returns a string representation of the response code
func (rc RCode) String() string {
	switch rc {
	case RCodeSuccess:
		return "NOERROR"
	case RCodeFormatError:
		return "FORMERR"
	case RCodeServerFailure:
		return "SERVFAIL"
	case RCodeNameError:
		return "NXDOMAIN"
	case RCodeNotImplemented:
		return "NOTIMP"
	case RCodeRefused:
		return "REFUSED"
	case RCodeNameExists:
		return "NAMEEXISTS"
	case RCodeRRSetExists:
		return "RRSEXISTS"
	case RCodeRRSetNotExists:
		return "RRNOTEXISTS"
	case RCodeNotAuth:
		return "NOTAUTH"
	case RCodeNotZone:
		return "NOTZONE"
	default:
		return "UNKNOWN"
	}
}

// Question represents a DNS question section
type Question struct {
	Name  string
	Type  RecordType
	Class RecordClass
}

// ResourceRecord represents a DNS resource record
type ResourceRecord struct {
	Name       string
	Type       RecordType
	Class      RecordClass
	TTL        time.Duration
	RDLength   uint16
	RData      []byte
	Expiration time.Time
}

// IP returns the IP address for A or AAAA records
func (rr *ResourceRecord) IP() net.IP {
	if rr.Type == RecordTypeA && len(rr.RData) == 4 {
		return net.IP(rr.RData)
	}
	if rr.Type == RecordTypeAAAA && len(rr.RData) == 16 {
		return net.IP(rr.RData)
	}
	return nil
}

// CNAME returns the canonical name for CNAME records
func (rr *ResourceRecord) CNAME() string {
	if rr.Type == RecordTypeCNAME {
		return string(rr.RData)
	}
	return ""
}

// MXPriority returns the priority for MX records
func (rr *ResourceRecord) MXPriority() uint16 {
	if rr.Type == RecordTypeMX && len(rr.RData) >= 2 {
		return uint16(rr.RData[0])<<8 | uint16(rr.RData[1])
	}
	return 0
}

// MXHost returns the mail exchange host for MX records
func (rr *ResourceRecord) MXHost() string {
	if rr.Type == RecordTypeMX && len(rr.RData) >= 2 {
		return string(rr.RData[2:])
	}
	return ""
}

// TXT returns the text content for TXT records
func (rr *ResourceRecord) TXT() string {
	if rr.Type == RecordTypeTXT {
		return string(rr.RData)
	}
	return ""
}

// NS returns the nameserver for NS records
func (rr *ResourceRecord) NS() string {
	if rr.Type == RecordTypeNS {
		return string(rr.RData)
	}
	return ""
}

// Message represents a complete DNS message
type Message struct {
	ID          uint16
	QR          bool   // Query (false) or Response (true)
	Opcode      Opcode // Operation code
	AA          bool   // Authoritative Answer
	TC          bool   // Truncation Flag
	RD          bool   // Recursion Desired
	RA          bool   // Recursion Available
	Z           uint8  // Reserved Z flags
	RCODE       RCode  // Response code
	Questions   []Question
	Answers     []ResourceRecord
	Authorities []ResourceRecord
	Extras      []ResourceRecord
}

// Header returns the DNS header flags as a uint16
func (m *Message) Header() uint16 {
	var flags uint16
	if m.QR {
		flags |= FlagQR
	}
	flags |= uint16(m.Opcode) << 11
	if m.AA {
		flags |= FlagAA
	}
	if m.TC {
		flags |= FlagTC
	}
	if m.RD {
		flags |= FlagRD
	}
	if m.RA {
		flags |= FlagRA
	}
	flags |= uint16(m.Z) << 4
	flags |= uint16(m.RCODE)
	return flags
}

// SetHeader sets the DNS header flags from a uint16
func (m *Message) SetHeader(flags uint16) {
	m.QR = (flags & FlagQR) != 0
	m.Opcode = Opcode((flags >> 11) & 0x0F)
	m.AA = (flags & FlagAA) != 0
	m.TC = (flags & FlagTC) != 0
	m.RD = (flags & FlagRD) != 0
	m.RA = (flags & FlagRA) != 0
	m.Z = uint8((flags >> 4) & 0x07)
	m.RCODE = RCode(flags & 0x0F)
}

// IsSuccess returns true if the response code indicates success
func (m *Message) IsSuccess() bool {
	return m.RCODE == RCodeSuccess
}

// IsNXDOMAIN returns true if the domain does not exist
func (m *Message) IsNXDOMAIN() bool {
	return m.RCODE == RCodeNameError
}
