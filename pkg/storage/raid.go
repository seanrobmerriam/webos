/*
Package storage provides block storage abstraction with caching,
write-ahead logging, RAID support, and snapshot management.
*/
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// RAID errors
var (
	ErrRAIDInvalidLevel   = errors.New("invalid RAID level")
	ErrRAIDDeviceCount    = errors.New("invalid number of devices")
	ErrRAIDDeviceFailed   = errors.New("device has failed")
	ErrRAIDRebuilding     = errors.New("RAID is rebuilding")
	ErrRAIDNotEnoughSpace = errors.New("not enough space for RAID layout")
)

// RAIDLevel represents the RAID configuration.
type RAIDLevel int

const (
	// RAIDLevel0 (striping) provides no redundancy.
	RAIDLevel0 RAIDLevel = iota
	// RAIDLevel1 (mirroring) provides full redundancy.
	RAIDLevel1
	// RAIDLevel5 (striping with parity) provides single-drive redundancy.
	RAIDLevel5
)

// RAIDStatus represents the current state of a RAID.
type RAIDStatus struct {
	Level           RAIDLevel `json:"level"`
	TotalDevices    int       `json:"totalDevices"`
	FailedDevices   int       `json:"failedDevices"`
	Rebuilding      bool      `json:"rebuilding"`
	RebuildProgress float64   `json:"rebuildProgress"`
	Capacity        uint64    `json:"capacity"`
	BlockSize       int       `json:"blockSize"`
	Healthy         bool      `json:"healthy"`
}

// RAID interface represents a RAID storage device.
type RAID interface {
	BlockDevice
	Rebuild(failedDevice int) error
	Status() RAIDStatus
	AddDevice(device BlockDevice, index int) error
	RemoveDevice(index int) error
	MarkDeviceFailed(index int) error
}

// BaseRAID provides common RAID functionality.
type BaseRAID struct {
	devices    []BlockDevice
	failed     []bool
	level      RAIDLevel
	blockSize  int
	totalSize  uint64
	mu         sync.RWMutex
	rebuilding int32
}

// newBaseRAID creates a new base RAID structure.
func newBaseRAID(devices []BlockDevice, level RAIDLevel) (*BaseRAID, error) {
	if len(devices) == 0 {
		return nil, ErrRAIDDeviceCount
	}

	// Verify all devices have same block size
	blockSize := devices[0].BlockSize()
	for _, device := range devices[1:] {
		if device.BlockSize() != blockSize {
			return nil, fmt.Errorf("device block size mismatch: %d != %d", device.BlockSize(), blockSize)
		}
	}

	r := &BaseRAID{
		devices:   devices,
		failed:    make([]bool, len(devices)),
		level:     level,
		blockSize: blockSize,
	}

	// Calculate capacity based on RAID level
	switch level {
	case RAIDLevel0:
		r.calculateRAID0Capacity()
	case RAIDLevel1:
		r.calculateRAID1Capacity()
	case RAIDLevel5:
		r.calculateRAID5Capacity()
	}

	return r, nil
}

// calculateRAID0Capacity calculates capacity for RAID 0.
func (r *BaseRAID) calculateRAID0Capacity() {
	var minBlocks uint64 = ^uint64(0)
	for _, device := range r.devices {
		if device.BlockCount() < minBlocks {
			minBlocks = device.BlockCount()
		}
	}
	r.totalSize = minBlocks * uint64(len(r.devices))
}

// calculateRAID1Capacity calculates capacity for RAID 1.
func (r *BaseRAID) calculateRAID1Capacity() {
	// Capacity is the size of the smallest device
	var minBlocks uint64 = ^uint64(0)
	for _, device := range r.devices {
		if device.BlockCount() < minBlocks {
			minBlocks = device.BlockCount()
		}
	}
	r.totalSize = minBlocks
}

// calculateRAID5Capacity calculates capacity for RAID 5.
func (r *BaseRAID) calculateRAID5Capacity() {
	// Capacity is (n-1) * min_block_count
	var minBlocks uint64 = ^uint64(0)
	for _, device := range r.devices {
		if device.BlockCount() < minBlocks {
			minBlocks = device.BlockCount()
		}
	}
	r.totalSize = minBlocks * uint64(len(r.devices)-1)
}

// BlockSize returns the block size.
func (r *BaseRAID) BlockSize() int {
	return r.blockSize
}

// BlockCount returns the total block count.
func (r *BaseRAID) BlockCount() uint64 {
	return r.totalSize
}

// IsHealthy checks if the RAID is healthy.
func (r *BaseRAID) IsHealthy() bool {
	count := 0
	for _, failed := range r.failed {
		if failed {
			count++
		}
	}

	switch r.level {
	case RAIDLevel0:
		return count == 0
	case RAIDLevel1:
		return count < len(r.devices)
	case RAIDLevel5:
		return count <= 1
	}
	return false
}

// GetStatus returns the current RAID status.
func (r *BaseRAID) GetStatus() RAIDStatus {
	count := 0
	for _, failed := range r.failed {
		if failed {
			count++
		}
	}

	rebuilding := atomic.LoadInt32(&r.rebuilding) == 1

	return RAIDStatus{
		Level:         r.level,
		TotalDevices:  len(r.devices),
		FailedDevices: count,
		Rebuilding:    rebuilding,
		Capacity:      r.totalSize,
		BlockSize:     r.blockSize,
		Healthy:       r.IsHealthy(),
	}
}

// RAID0Stripe implements striping without redundancy.
type RAID0Stripe struct {
	*BaseRAID
}

// NewRAID0 creates a new RAID 0 array.
func NewRAID0(devices []BlockDevice) (*RAID0Stripe, error) {
	if len(devices) < 2 {
		return nil, ErrRAIDDeviceCount
	}

	base, err := newBaseRAID(devices, RAIDLevel0)
	if err != nil {
		return nil, err
	}

	return &RAID0Stripe{BaseRAID: base}, nil
}

// Read reads from RAID 0.
func (r *RAID0Stripe) Read(block uint64, data []byte) error {
	if uint64(len(data)) != uint64(r.blockSize) {
		return fmt.Errorf("data size mismatch: %d != %d", len(data), r.blockSize)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Calculate which device and offset
	deviceIdx := int(block % uint64(len(r.devices)))
	localBlock := block / uint64(len(r.devices))

	// Check if device failed
	if r.failed[deviceIdx] {
		return ErrRAIDDeviceFailed
	}

	return r.devices[deviceIdx].Read(localBlock, data)
}

// Write writes to RAID 0.
func (r *RAID0Stripe) Write(block uint64, data []byte) error {
	if uint64(len(data)) != uint64(r.blockSize) {
		return ErrBlockTooLarge
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Calculate which device and offset
	deviceIdx := int(block % uint64(len(r.devices)))
	localBlock := block / uint64(len(r.devices))

	// Check if device failed
	if r.failed[deviceIdx] {
		return ErrRAIDDeviceFailed
	}

	return r.devices[deviceIdx].Write(localBlock, data)
}

// Flush flushes all devices.
func (r *RAID0Stripe) Flush() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i, device := range r.devices {
		if !r.failed[i] {
			device.Flush()
		}
	}
	return nil
}

// Status returns the current RAID status.
func (r *RAID0Stripe) Status() RAIDStatus {
	return r.GetStatus()
}

// Close closes all devices.
func (r *RAID0Stripe) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, device := range r.devices {
		if !r.failed[i] {
			device.Close()
		}
	}
	return nil
}

// Rebuild is a no-op for RAID 0 (no redundancy).
func (r *RAID0Stripe) Rebuild(failedDevice int) error {
	return ErrRAIDNotEnoughSpace
}

// AddDevice adds a device to the array (expands RAID 0).
func (r *RAID0Stripe) AddDevice(device BlockDevice, index int) error {
	return ErrNotSupported
}

// RemoveDevice removes a device from the array.
func (r *RAID0Stripe) RemoveDevice(index int) error {
	return ErrNotSupported
}

// MarkDeviceFailed marks a device as failed.
func (r *RAID0Stripe) MarkDeviceFailed(index int) error {
	return ErrNotSupported
}

// RAID1Mirror implements full mirroring.
type RAID1Mirror struct {
	*BaseRAID
}

// NewRAID1 creates a new RAID 1 array.
func NewRAID1(devices []BlockDevice) (*RAID1Mirror, error) {
	if len(devices) < 2 {
		return nil, ErrRAIDDeviceCount
	}

	base, err := newBaseRAID(devices, RAIDLevel1)
	if err != nil {
		return nil, err
	}

	return &RAID1Mirror{BaseRAID: base}, nil
}

// Read reads from all mirrors and returns the first successful.
func (r *RAID1Mirror) Read(block uint64, data []byte) error {
	if uint64(len(data)) != uint64(r.blockSize) {
		return fmt.Errorf("data size mismatch: %d != %d", len(data), r.blockSize)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try each device in order
	for i, device := range r.devices {
		if r.failed[i] {
			continue
		}
		if err := device.Read(block, data); err == nil {
			return nil
		}
	}

	return ErrRAIDDeviceFailed
}

// Write writes to all mirrors.
func (r *RAID1Mirror) Write(block uint64, data []byte) error {
	if uint64(len(data)) != uint64(r.blockSize) {
		return ErrBlockTooLarge
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Write to all non-failed devices
	for i, device := range r.devices {
		if r.failed[i] {
			continue
		}
		if err := device.Write(block, data); err != nil {
			return err
		}
	}

	return nil
}

// Flush flushes all devices.
func (r *RAID1Mirror) Flush() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i, device := range r.devices {
		if !r.failed[i] {
			device.Flush()
		}
	}
	return nil
}

// Close closes all devices.
func (r *RAID1Mirror) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, device := range r.devices {
		if !r.failed[i] {
			device.Close()
		}
	}
	return nil
}

// Status returns the current RAID status.
func (r *RAID1Mirror) Status() RAIDStatus {
	return r.GetStatus()
}

// Rebuild reconstructs data on a failed device.
func (r *RAID1Mirror) Rebuild(failedDevice int) error {
	if failedDevice < 0 || failedDevice >= len(r.devices) {
		return fmt.Errorf("invalid device index: %d", failedDevice)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.failed[failedDevice] {
		return nil // Device is not failed
	}

	// Find a working device
	var sourceDevice int = -1
	for i := range r.devices {
		if i != failedDevice && !r.failed[i] {
			sourceDevice = i
			break
		}
	}

	if sourceDevice == -1 {
		return ErrRAIDDeviceFailed
	}

	// Read from source and write to failed device
	blockCount := r.devices[sourceDevice].BlockCount()
	data := make([]byte, r.blockSize)

	for block := uint64(0); block < blockCount; block++ {
		if err := r.devices[sourceDevice].Read(block, data); err != nil {
			return err
		}
		if err := r.devices[failedDevice].Write(block, data); err != nil {
			return err
		}
	}

	r.failed[failedDevice] = false
	return nil
}

// AddDevice adds a mirror device.
func (r *RAID1Mirror) AddDevice(device BlockDevice, index int) error {
	return ErrNotSupported
}

// RemoveDevice removes a mirror device.
func (r *RAID1Mirror) RemoveDevice(index int) error {
	return ErrNotSupported
}

// MarkDeviceFailed marks a device as failed.
func (r *RAID1Mirror) MarkDeviceFailed(index int) error {
	if index < 0 || index >= len(r.devices) {
		return fmt.Errorf("invalid device index: %d", index)
	}
	r.failed[index] = true
	return nil
}

// RAID5Parity implements striping with distributed parity.
type RAID5Parity struct {
	*BaseRAID
	parityBlock uint64 // Current parity block position
}

// NewRAID5 creates a new RAID 5 array.
func NewRAID5(devices []BlockDevice) (*RAID5Parity, error) {
	if len(devices) < 3 {
		return nil, ErrRAIDDeviceCount
	}

	base, err := newBaseRAID(devices, RAIDLevel5)
	if err != nil {
		return nil, err
	}

	return &RAID5Parity{
		BaseRAID:    base,
		parityBlock: 0,
	}, nil
}

// Read reads from RAID 5.
func (r *RAID5Parity) Read(block uint64, data []byte) error {
	if uint64(len(data)) != uint64(r.blockSize) {
		return fmt.Errorf("data size mismatch: %d != %d", len(data), r.blockSize)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Calculate stripe info
	stripeSize := uint64(len(r.devices) - 1)
	stripe := block / stripeSize
	offset := block % stripeSize

	// Determine data and parity device indices
	dataIdx := int(offset)
	parityIdx := int(stripe % uint64(len(r.devices)))

	// Adjust data index to skip parity
	if dataIdx >= parityIdx {
		dataIdx++
	}

	// Check for failed devices
	failedIdx := -1
	for i, failed := range r.failed {
		if failed {
			failedIdx = i
			break
		}
	}

	if failedIdx == -1 {
		// All devices healthy, read from data device
		return r.devices[dataIdx].Read(stripe, data)
	}

	// Need to reconstruct
	return r.reconstructRead(stripe, dataIdx, parityIdx, failedIdx, data)
}

// reconstructRead rebuilds data from remaining devices.
func (r *RAID5Parity) reconstructRead(stripe uint64, dataIdx, parityIdx, failedIdx int, data []byte) error {
	// Collect data from all non-failed devices
	parityData := make([]byte, r.blockSize)
	dataDevices := make([][]byte, len(r.devices)-1)
	dataIndices := make([]int, len(r.devices)-1)

	idx := 0
	for i, device := range r.devices {
		if i == failedIdx {
			continue
		}
		if i == parityIdx {
			device.Read(stripe, parityData)
		} else {
			buf := make([]byte, r.blockSize)
			device.Read(stripe, buf)
			dataDevices[idx] = buf
			dataIndices[idx] = i
			idx++
		}
	}

	// XOR to reconstruct missing data
	xorResult := make([]byte, r.blockSize)
	for _, devData := range dataDevices {
		for j := 0; j < len(xorResult); j++ {
			xorResult[j] ^= devData[j]
		}
	}

	if failedIdx == parityIdx {
		// Parity was lost
		copy(data, xorResult)
	} else {
		// Data was lost
		for j := 0; j < len(xorResult); j++ {
			xorResult[j] ^= parityData[j]
		}
		copy(data, xorResult)
	}

	return nil
}

// Write writes to RAID 5.
func (r *RAID5Parity) Write(block uint64, data []byte) error {
	if uint64(len(data)) != uint64(r.blockSize) {
		return ErrBlockTooLarge
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Calculate stripe info
	stripeSize := uint64(len(r.devices) - 1)
	stripe := block / stripeSize
	offset := block % stripeSize

	// Determine data and parity device indices
	dataIdx := int(offset)
	parityIdx := int(stripe % uint64(len(r.devices)))
	if dataIdx >= parityIdx {
		dataIdx++
	}

	// Check for failed devices
	failedCount := 0
	for _, failed := range r.failed {
		if failed {
			failedCount++
		}
	}

	if failedCount > 0 {
		return ErrRAIDDeviceFailed
	}

	// Read old data and parity
	oldData := make([]byte, r.blockSize)
	oldParity := make([]byte, r.blockSize)

	r.devices[dataIdx].Read(stripe, oldData)
	r.devices[parityIdx].Read(stripe, oldParity)

	// Calculate new parity: newParity = oldParity XOR oldData XOR newData
	newParity := make([]byte, r.blockSize)
	for i := 0; i < len(newParity); i++ {
		newParity[i] = oldParity[i] ^ oldData[i] ^ data[i]
	}

	// Write new data and parity
	if err := r.devices[dataIdx].Write(stripe, data); err != nil {
		return err
	}
	if err := r.devices[parityIdx].Write(stripe, newParity); err != nil {
		return err
	}

	return nil
}

// Flush flushes all devices.
func (r *RAID5Parity) Flush() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i, device := range r.devices {
		if !r.failed[i] {
			device.Flush()
		}
	}
	return nil
}

// Close closes all devices.
func (r *RAID5Parity) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, device := range r.devices {
		if !r.failed[i] {
			device.Close()
		}
	}
	return nil
}

// Status returns the current RAID status.
func (r *RAID5Parity) Status() RAIDStatus {
	return r.GetStatus()
}

// Rebuild reconstructs a failed device.
func (r *RAID5Parity) Rebuild(failedDevice int) error {
	if failedDevice < 0 || failedDevice >= len(r.devices) {
		return fmt.Errorf("invalid device index: %d", failedDevice)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.failed[failedDevice] {
		return nil
	}

	atomic.StoreInt32(&r.rebuilding, 1)
	defer atomic.StoreInt32(&r.rebuilding, 0)

	// Rebuild stripe by stripe
	stripeCount := r.devices[0].BlockCount()
	data := make([]byte, r.blockSize)

	for stripe := uint64(0); stripe < stripeCount; stripe++ {
		// XOR all data to reconstruct
		xorData := make([]byte, r.blockSize)
		for i, device := range r.devices {
			if i == failedDevice || r.failed[i] {
				continue
			}
			device.Read(stripe, data)
			for j := 0; j < len(xorData); j++ {
				xorData[j] ^= data[j]
			}
		}

		// Write to failed device
		if err := r.devices[failedDevice].Write(stripe, xorData); err != nil {
			return err
		}
	}

	r.failed[failedDevice] = false
	return nil
}

// AddDevice adds a device (requires reshape).
func (r *RAID5Parity) AddDevice(device BlockDevice, index int) error {
	return ErrNotSupported
}

// RemoveDevice removes a device.
func (r *RAID5Parity) RemoveDevice(index int) error {
	return ErrNotSupported
}

// MarkDeviceFailed marks a device as failed.
func (r *RAID5Parity) MarkDeviceFailed(index int) error {
	if index < 0 || index >= len(r.devices) {
		return fmt.Errorf("invalid device index: %d", index)
	}
	r.failed[index] = true
	return nil
}

// RAIDFactory creates RAID arrays.
type RAIDFactory struct{}

// NewRAIDFactory creates a new RAID factory.
func NewRAIDFactory() *RAIDFactory {
	return &RAIDFactory{}
}

// CreateRAID creates a RAID array of the specified level.
func (f *RAIDFactory) CreateRAID(devices []BlockDevice, level RAIDLevel) (RAID, error) {
	switch level {
	case RAIDLevel0:
		return NewRAID0(devices)
	case RAIDLevel1:
		return NewRAID1(devices)
	case RAIDLevel5:
		return NewRAID5(devices)
	default:
		return nil, ErrRAIDInvalidLevel
	}
}

// RAIDStats contains RAID statistics.
type RAIDStats struct {
	Status        RAIDStatus
	Devices       []DeviceInfo
	Configuration json.RawMessage
}

// GetStats returns RAID statistics.
func GetStats(raid RAID) RAIDStats {
	status := raid.Status()

	devices := make([]DeviceInfo, 0)
	// Collect device info would go here

	return RAIDStats{
		Status:  status,
		Devices: devices,
	}
}
