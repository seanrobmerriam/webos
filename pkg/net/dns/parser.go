package dns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

var (
	// ErrInvalidMessage indicates the DNS message is malformed
	ErrInvalidMessage = errors.New("invalid DNS message")
	// ErrTruncated indicates the response was truncated
	ErrTruncated = errors.New("DNS response truncated")
	// ErrCompression indicates invalid name compression
	ErrCompression = errors.New("invalid name compression")

	// idCounter is used to generate unique DNS message IDs
	idCounter uint32
)

// Parser handles DNS message parsing and serialization
type Parser struct {
	buf   []byte
	pos   int
	off   int // offset for message compression
	bytes int // total bytes in message
}

// NewParser creates a new DNS parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseMessage parses a DNS message from raw bytes
func (p *Parser) ParseMessage(data []byte) (*Message, error) {
	if len(data) < 12 {
		return nil, ErrInvalidMessage
	}

	p.buf = data
	p.pos = 0
	p.off = 12
	p.bytes = len(data)

	// Parse header
	id := binary.BigEndian.Uint16(data[0:2])
	flags := binary.BigEndian.Uint16(data[2:4])
	qdcount := binary.BigEndian.Uint16(data[4:6])
	ancount := binary.BigEndian.Uint16(data[6:8])
	nscount := binary.BigEndian.Uint16(data[8:10])
	arcount := binary.BigEndian.Uint16(data[10:12])

	m := &Message{
		ID:        id,
		Questions: make([]Question, qdcount),
		Answers:   make([]ResourceRecord, ancount),
	}
	m.SetHeader(flags)

	// Store positions for compression
	positions := make(map[int]string)

	p.pos = 12

	// Parse questions
	for i := uint16(0); i < qdcount; i++ {
		name, err := p.parseName(positions, false)
		if err != nil {
			return nil, fmt.Errorf("%w: question %d: %v", ErrInvalidMessage, i, err)
		}
		qt := binary.BigEndian.Uint16(p.buf[p.pos : p.pos+2])
		qc := binary.BigEndian.Uint16(p.buf[p.pos+2 : p.pos+4])
		p.pos += 4

		m.Questions[i] = Question{
			Name:  name,
			Type:  RecordType(qt),
			Class: RecordClass(qc),
		}
	}

	// Parse answer records
	for i := uint16(0); i < ancount; i++ {
		rr, err := p.parseResourceRecord(positions)
		if err != nil {
			return nil, fmt.Errorf("%w: answer %d: %v", ErrInvalidMessage, i, err)
		}
		m.Answers[i] = *rr
	}

	// Parse authority records
	for i := uint16(0); i < nscount; i++ {
		rr, err := p.parseResourceRecord(positions)
		if err != nil {
			return nil, fmt.Errorf("%w: authority %d: %v", ErrInvalidMessage, i, err)
		}
		m.Authorities = append(m.Authorities, *rr)
	}

	// Parse extra records
	for i := uint16(0); i < arcount; i++ {
		rr, err := p.parseResourceRecord(positions)
		if err != nil {
			return nil, fmt.Errorf("%w: extra record: %v", ErrInvalidMessage, err)
		}
		m.Extras = append(m.Extras, *rr)
	}

	// Check for truncation
	if m.TC {
		return m, ErrTruncated
	}

	return m, nil
}

// parseName parses a domain name with compression support
func (p *Parser) parseName(positions map[int]string, allowCompression bool) (string, error) {
	var name bytes.Buffer
	loop := 0

	for loop < 256 {
		if p.pos >= len(p.buf) {
			return "", ErrInvalidMessage
		}

		length := int(p.buf[p.pos])
		p.pos++

		// Check for compression pointer
		if allowCompression && (length&0xC0) == 0xC0 {
			if p.pos >= len(p.buf) {
				return "", ErrCompression
			}
			pointer := int(length&0x3F)<<8 | int(p.buf[p.pos])
			p.pos++

			if pointer >= p.off {
				return "", ErrCompression
			}

			// Resolve pointer recursively
			oldPos := p.pos
			oldOff := p.off
			p.pos = pointer

			pointedName, err := p.parseName(positions, false)
			if err != nil {
				return "", err
			}

			p.pos = oldPos
			p.off = oldOff

			if name.Len() > 0 {
				name.WriteByte('.')
			}
			name.WriteString(pointedName)
			break
		}

		// End of name
		if length == 0 {
			break
		}

		if p.pos+length > len(p.buf) {
			return "", ErrInvalidMessage
		}

		if name.Len() > 0 {
			name.WriteByte('.')
		}
		name.Write(p.buf[p.pos : p.pos+length])
		p.pos += length

		// Record position for compression (but not for the root label)
		if p.off < len(p.buf) && length > 0 {
			positions[p.off] = name.String()
			p.off++
		}

		loop++
	}

	if loop >= 256 {
		return "", ErrInvalidMessage
	}

	return name.String(), nil
}

// parseResourceRecord parses a DNS resource record
func (p *Parser) parseResourceRecord(positions map[int]string) (*ResourceRecord, error) {
	if p.pos >= len(p.buf) {
		return nil, ErrInvalidMessage
	}

	name, err := p.parseName(positions, true)
	if err != nil {
		return nil, err
	}

	if p.pos+10 > len(p.buf) {
		return nil, ErrInvalidMessage
	}

	rdlength := binary.BigEndian.Uint16(p.buf[p.pos+8 : p.pos+10])
	rdstart := p.pos + 10

	if rdstart+int(rdlength) > len(p.buf) {
		return nil, ErrInvalidMessage
	}

	rr := &ResourceRecord{
		Name:     name,
		Type:     RecordType(binary.BigEndian.Uint16(p.buf[p.pos : p.pos+2])),
		Class:    RecordClass(binary.BigEndian.Uint16(p.buf[p.pos+2 : p.pos+4])),
		TTL:      time.Duration(binary.BigEndian.Uint32(p.buf[p.pos+4:p.pos+8])) * time.Second,
		RDLength: rdlength,
		RData:    make([]byte, rdlength),
	}
	copy(rr.RData, p.buf[rdstart:rdstart+int(rdlength)])

	// Set expiration time
	if rr.TTL > 0 {
		rr.Expiration = time.Now().Add(rr.TTL)
	}

	p.pos = rdstart + int(rdlength)
	return rr, nil
}

// BuildMessage builds a DNS message from a Message struct
func (p *Parser) BuildMessage(m *Message) ([]byte, error) {
	// Estimate size and build
	buf := &bytes.Buffer{}

	// Write ID
	if err := binary.Write(buf, binary.BigEndian, m.ID); err != nil {
		return nil, err
	}

	// Write flags
	if err := binary.Write(buf, binary.BigEndian, m.Header()); err != nil {
		return nil, err
	}

	// Write counts
	qdcount := uint16(len(m.Questions))
	ancount := uint16(len(m.Answers))
	nscount := uint16(len(m.Authorities))
	arcount := uint16(len(m.Extras))

	// Write QDCOUNT
	if err := binary.Write(buf, binary.BigEndian, qdcount); err != nil {
		return nil, err
	}

	// Save position for ANCOUNT (offset 6)
	ancountPos := buf.Len()

	// Write ANCOUNT placeholder
	if err := binary.Write(buf, binary.BigEndian, uint16(0)); err != nil {
		return nil, err
	}

	// Write NSCOUNT
	if err := binary.Write(buf, binary.BigEndian, nscount); err != nil {
		return nil, err
	}

	// Write ARCOUNT
	if err := binary.Write(buf, binary.BigEndian, arcount); err != nil {
		return nil, err
	}

	// Track positions for compression
	positions := make(map[int]string)

	// Write questions
	for _, q := range m.Questions {
		if err := p.writeName(buf, q.Name, positions); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(q.Type)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(q.Class)); err != nil {
			return nil, err
		}
	}

	// Write answers
	for _, rr := range m.Answers {
		if err := p.writeName(buf, rr.Name, positions); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(rr.Type)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(rr.Class)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint32(rr.TTL/time.Second)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, rr.RDLength); err != nil {
			return nil, err
		}
		if _, err := buf.Write(rr.RData); err != nil {
			return nil, err
		}
	}

	// Write authorities
	for _, rr := range m.Authorities {
		if err := p.writeName(buf, rr.Name, positions); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(rr.Type)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(rr.Class)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint32(rr.TTL/time.Second)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, rr.RDLength); err != nil {
			return nil, err
		}
		if _, err := buf.Write(rr.RData); err != nil {
			return nil, err
		}
	}

	// Write extras
	for _, rr := range m.Extras {
		if err := p.writeName(buf, rr.Name, positions); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(rr.Type)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(rr.Class)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, uint32(rr.TTL/time.Second)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, rr.RDLength); err != nil {
			return nil, err
		}
		if _, err := buf.Write(rr.RData); err != nil {
			return nil, err
		}
	}

	// Update ANCOUNT at the correct position
	data := buf.Bytes()
	binary.BigEndian.PutUint16(data[ancountPos:], ancount)

	return data, nil
}

// writeName writes a domain name to DNS wire format
func (p *Parser) writeName(buf *bytes.Buffer, name string, positions map[int]string) error {
	labels := strings.Split(name, ".")

	for i, label := range labels {
		length := len(label)

		// Write label length
		if err := buf.WriteByte(byte(length)); err != nil {
			return err
		}

		// Write label content
		if _, err := buf.WriteString(label); err != nil {
			return err
		}

		// Record position for this label (for future compression)
		// Position is where the length byte was written
		pos := buf.Len() - length - 1
		if pos >= 12 { // Only record positions after the header
			positions[pos] = name
		}

		// Don't add trailing dot for last label
		if i < len(labels)-1 {
			if err := buf.WriteByte('.'); err != nil {
				return err
			}
		}
	}

	// Write null terminator
	return buf.WriteByte(0)
}

// BuildQuery creates a DNS query message for the given name and type
func BuildQuery(name string, qtype RecordType) (*Message, error) {
	return &Message{
		ID:     generateID(),
		QR:     false,
		Opcode: OpcodeQuery,
		RD:     true,
		Questions: []Question{
			{
				Name:  name,
				Type:  qtype,
				Class: ClassIN,
			},
		},
	}, nil
}

// BuildAXFRQuery creates a DNS query for AXFR (zone transfer)
func BuildAXFRQuery(name string) (*Message, error) {
	return &Message{
		ID:     generateID(),
		QR:     false,
		Opcode: OpcodeQuery,
		RD:     true,
		Questions: []Question{
			{
				Name:  name,
				Type:  RecordType(RecordTypeAXFR),
				Class: ClassIN,
			},
		},
	}, nil
}

// RecordTypeAXFR is the AXFR record type for zone transfers
const RecordTypeAXFR uint16 = 252

// generateID generates a unique 16-bit ID for DNS messages
func generateID() uint16 {
	return uint16(atomic.AddUint32(&idCounter, 1) & 0xFFFF)
}

// ParseIP parses an IP address from resource record data
func ParseIP(rr *ResourceRecord) net.IP {
	if rr.Type == RecordTypeA && len(rr.RData) == 4 {
		return net.IP(rr.RData)
	}
	if rr.Type == RecordTypeAAAA && len(rr.RData) == 16 {
		return net.IP(rr.RData)
	}
	return nil
}
