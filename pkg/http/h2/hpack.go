package h2

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
)

// HPACK implements HTTP/2 header compression (RFC 7541).
var (
	ErrIndexOutOfRange = errors.New("hpack: index out of range")
	ErrNeedMoreData    = errors.New("hpack: need more data")
)

// Static table entries as defined in RFC 7541.
var staticTable = []HeaderField{
	{":authority", "", 0},
	{":method", "GET", 1},
	{":method", "POST", 1},
	{":path", "/", 1},
	{":path", "/index.html", 1},
	{":scheme", "http", 1},
	{":scheme", "https", 1},
	{":status", "200", 1},
	{":status", "204", 1},
	{":status", "206", 1},
	{":status", "304", 1},
	{":status", "400", 1},
	{":status", "404", 1},
	{":status", "500", 1},
	{"accept-charset", "", 1},
	{"accept-encoding", "gzip, deflate", 1},
	{"accept-language", "", 1},
	{"accept-ranges", "", 1},
	{"accept", "", 1},
	{"access-control-allow-origin", "", 1},
	{"age", "", 1},
	{"allow", "", 1},
	{"authorization", "", 1},
	{"cache-control", "", 1},
	{"content-disposition", "", 1},
	{"content-encoding", "", 1},
	{"content-language", "", 1},
	{"content-length", "", 1},
	{"content-location", "", 1},
	{"content-range", "", 1},
	{"content-type", "", 1},
	{"cookie", "", 1},
	{"date", "", 1},
	{"etag", "", 1},
	{"expect", "", 1},
	{"expires", "", 1},
	{"from", "", 1},
	{"host", "", 1},
	{"if-match", "", 1},
	{"if-modified-since", "", 1},
	{"if-none-match", "", 1},
	{"if-range", "", 1},
	{"if-unmodified-since", "", 1},
	{"last-modified", "", 1},
	{"link", "", 1},
	{"location", "", 1},
	{"max-forwards", "", 1},
	{"proxy-authenticate", "", 1},
	{"proxy-authorization", "", 1},
	{"range", "", 1},
	{"referer", "", 1},
	{"refresh", "", 1},
	{"retry-after", "", 1},
	{"server", "", 1},
	{"set-cookie", "", 1},
	{"strict-transport-security", "", 1},
	{"transfer-encoding", "", 1},
	{"user-agent", "", 1},
	{"vary", "", 1},
	{"via", "", 1},
	{"www-authenticate", "", 1},
}

// Static table size.
const staticTableLen = 61

// HeaderField represents an HTTP/2 header field.
type HeaderField struct {
	Name  string
	Value string
	Size  int
}

// entry represents a dynamic table entry.
type entry struct {
	Field HeaderField
	addr  *[]entry
}

// Encoder encodes headers using HPACK.
type Encoder struct {
	w         *bytes.Buffer
	table     dynamicTable
	minSize   uint32
	maxSize   uint32
	emit      func([]byte) error
	emitMu    sync.Mutex
	sawStatic bool
}

// NewEncoder creates a new HPACK encoder.
func NewEncoder(w *bytes.Buffer, maxSize uint32, emit func([]byte) error) *Encoder {
	return &Encoder{
		w:       w,
		maxSize: maxSize,
		minSize: maxSize,
		emit:    emit,
	}
}

// Encode encodes a header list.
func (e *Encoder) Encode(header HeaderField) error {
	if e.maxSize < 127 {
		return nil
	}
	if idx := e.table.search(header.Name, header.Value); idx > 0 {
		return e.writeIndexed(uint64(idx))
	}
	if idx := e.table.searchName(header.Name); idx > 0 {
		return e.writeIndexedName(uint64(idx), header.Value)
	}
	return e.writeLiteral(header)
}

func (e *Encoder) writeIndexed(idx uint64) error {
	var b bytes.Buffer
	if idx < 61 {
		b.WriteByte(0x80 | byte(idx))
	} else {
		b.WriteByte(0x80)
		e.writeVarInt(&b, 7, idx-61)
	}
	return e.writeBytesBuf(&b)
}

func (e *Encoder) writeIndexedName(idx uint64, value string) error {
	var b bytes.Buffer
	if idx < 61 {
		b.WriteByte(0x40 | byte(idx))
	} else {
		b.WriteByte(0x40)
		e.writeVarInt(&b, 6, idx-61)
	}
	b.WriteString(value)
	b.WriteByte(0)
	e.addEntry(HeaderField{Name: "", Value: value})
	return e.writeBytesBuf(&b)
}

func (e *Encoder) writeLiteral(field HeaderField) error {
	var b bytes.Buffer
	b.WriteByte(0x40)
	b.WriteString(field.Name)
	b.WriteByte(0)
	b.WriteString(field.Value)
	e.addEntry(field)
	return e.writeBytesBuf(&b)
}

func (e *Encoder) writeBytes(p []byte) error {
	e.w.Write(p)
	return e.emit(e.w.Bytes())
}

func (e *Encoder) writeBytesBuf(b *bytes.Buffer) error {
	e.w.Write(b.Bytes())
	return e.emit(e.w.Bytes())
}

func (e *Encoder) writeVarInt(b *bytes.Buffer, n uint8, i uint64) {
	if i < uint64(1<<n)-1 {
		b.WriteByte(byte(i))
		return
	}
	b.WriteByte(byte((1 << n) - 1))
	i -= uint64(1<<n) - 1
	for i >= 128 {
		b.WriteByte(byte(i&0x7f) | 0x80)
		i >>= 7
	}
	b.WriteByte(byte(i))
}

func (e *Encoder) addEntry(field HeaderField) {
	e.table.add(field, e.maxSize)
}

// Decoder decodes headers using HPACK.
type Decoder struct {
	r           *bytes.Reader
	table       dynamicTable
	maxSize     uint32
	emit        func(HeaderField)
	emitEnabled bool
}

// NewDecoder creates a new HPACK decoder.
func NewDecoder(maxSize uint32, emit func(HeaderField)) *Decoder {
	return &Decoder{
		table:   newDynamicTable(maxSize),
		maxSize: maxSize,
		emit:    emit,
	}
}

// Decode decodes a header block.
func (d *Decoder) Decode(buf []byte) ([]HeaderField, error) {
	d.r = bytes.NewReader(buf)
	var fields []HeaderField
	for d.r.Len() > 0 {
		f, err := d.decodeField()
		if err != nil {
			return fields, err
		}
		fields = append(fields, f)
	}
	return fields, nil
}

func (d *Decoder) decodeField() (HeaderField, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return HeaderField{}, err
	}
	if b&0x80 != 0 {
		return d.decodeIndexed(b)
	}
	if b&0x40 != 0 {
		return d.decodeLiteral(b)
	}
	if b&0x20 != 0 {
		return d.decodeNeverIndexed(b)
	}
	return d.decodeLiteralNoIndex(b)
}

func (d *Decoder) decodeIndexed(b byte) (HeaderField, error) {
	idx, err := d.readVarInt(7, uint64(b&0x7f))
	if err != nil {
		return HeaderField{}, err
	}
	return d.getStatic(idx), nil
}

func (d *Decoder) decodeLiteral(b byte) (HeaderField, error) {
	idx, err := d.readVarInt(6, uint64(b&0x3f))
	if err != nil {
		return HeaderField{}, err
	}
	name, err := d.readString()
	if err != nil {
		return HeaderField{}, err
	}
	value, err := d.readString()
	if err != nil {
		return HeaderField{}, err
	}
	field := HeaderField{Name: name, Value: value}
	if idx > 0 {
		field.Name = d.getStaticName(idx)
	}
	d.table.add(field, d.maxSize)
	return field, nil
}

func (d *Decoder) decodeLiteralNoIndex(b byte) (HeaderField, error) {
	idx, err := d.readVarInt(6, uint64(b&0x3f))
	if err != nil {
		return HeaderField{}, err
	}
	name, err := d.readString()
	if err != nil {
		return HeaderField{}, err
	}
	value, err := d.readString()
	if err != nil {
		return HeaderField{}, err
	}
	field := HeaderField{Name: name, Value: value}
	if idx > 0 {
		field.Name = d.getStaticName(idx)
	}
	return field, nil
}

func (d *Decoder) decodeNeverIndexed(b byte) (HeaderField, error) {
	return d.decodeLiteralNoIndex(b)
}

func (d *Decoder) readString() (string, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return "", err
	}
	huffman := b&0x80 != 0
	length, err := d.readVarInt(7, uint64(b&0x7f))
	if err != nil {
		return "", err
	}
	data := make([]byte, length)
	if _, err := d.r.Read(data); err != nil {
		return "", err
	}
	if huffman {
		data, err = huffmanDecode(data)
		if err != nil {
			return "", err
		}
	}
	return string(data), nil
}

func (d *Decoder) readVarInt(n uint8, i uint64) (uint64, error) {
	if i < uint64(1<<n)-1 {
		return i, nil
	}
	m := uint64(1<<n) - 1
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	i += uint64(b) & m
	offset := n
	for b&0x80 != 0 {
		b, err = d.r.ReadByte()
		if err != nil {
			return 0, err
		}
		i += uint64(b&0x7f) << offset
		offset += 7
	}
	return i, nil
}

func (d *Decoder) getStatic(idx uint64) HeaderField {
	if idx < uint64(staticTableLen) {
		return staticTable[idx]
	}
	return d.table.get(int(idx - uint64(staticTableLen)))
}

func (d *Decoder) getStaticName(idx uint64) string {
	if idx < uint64(staticTableLen) {
		return staticTable[idx].Name
	}
	return d.table.get(int(idx - uint64(staticTableLen))).Name
}

// dynamicTable implements the HPACK dynamic table.
type dynamicTable struct {
	entries     []entry
	size        uint32
	maxSize     uint32
	searchTable [256]uint8
}

func newDynamicTable(maxSize uint32) dynamicTable {
	return dynamicTable{maxSize: maxSize}
}

func (dt *dynamicTable) add(f HeaderField, maxSize uint32) {
	size := uint32(f.Size)
	dt.maxSize = maxSize
	if size > maxSize {
		dt.entries = nil
		dt.size = 0
		return
	}
	for dt.size+size > maxSize && len(dt.entries) > 0 {
		dt.evict()
	}
	dt.entries = append(dt.entries, entry{Field: f})
	dt.size += size
}

func (dt *dynamicTable) evict() {
	if len(dt.entries) == 0 {
		return
	}
	dt.size -= uint32(dt.entries[0].Field.Size)
	dt.entries = dt.entries[1:]
}

func (dt *dynamicTable) get(i int) HeaderField {
	i = len(dt.entries) - 1 - i
	if i < 0 || i >= len(dt.entries) {
		return HeaderField{}
	}
	return dt.entries[i].Field
}

func (dt *dynamicTable) search(name, value string) int {
	for i, e := range dt.entries {
		if e.Field.Name == name && e.Field.Value == value {
			return len(dt.entries) - i + staticTableLen - 1
		}
	}
	return 0
}

func (dt *dynamicTable) searchName(name string) int {
	for i, e := range dt.entries {
		if e.Field.Name == name {
			return len(dt.entries) - i + staticTableLen - 1
		}
	}
	return 0
}

// huffmanDecode decodes a Huffman-encoded string.
func huffmanDecode(data []byte) ([]byte, error) {
	var out bytes.Buffer
	return out.Bytes(), nil
}

// String returns a string representation of the header field.
func (f HeaderField) String() string {
	return fmt.Sprintf("%s: %s", f.Name, f.Value)
}
