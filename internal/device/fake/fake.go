package fake

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/0xzahed/ventox/internal/domain"
)

const blockSize = 64 * 1024 // 64 KiB sparse block size

// SimulatedDisk provides a sparse memory-backed block device for testing partition and write operations
// with deterministic fault injection (short writes, sudden disconnects, readback verification).
type SimulatedDisk struct {
	mu           sync.RWMutex
	capacity     int64
	blocks       map[int64][]byte
	cursor       int64
	bytesWritten int64
	closed       bool

	// Fault injection parameters
	FailOnWriteAfterBytes int64
	SimulateShortWrite    bool
	SimulateDisconnect    bool
	InjectedError         error
}

// NewSimulatedDisk creates a simulated disk with a given capacity in bytes using sparse allocation.
func NewSimulatedDisk(capacityBytes int64) *SimulatedDisk {
	return &SimulatedDisk{
		capacity: capacityBytes,
		blocks:   make(map[int64][]byte),
	}
}

func (s *SimulatedDisk) Capacity() int64 {
	return s.capacity
}

func (s *SimulatedDisk) Read(p []byte) (n int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed || s.SimulateDisconnect {
		return 0, domain.ErrDeviceRemoved
	}
	if s.cursor >= s.capacity {
		return 0, io.EOF
	}

	readLen := int64(len(p))
	if s.cursor+readLen > s.capacity {
		readLen = s.capacity - s.cursor
	}

	for i := int64(0); i < readLen; i++ {
		currPos := s.cursor + i
		blockIdx := currPos / blockSize
		blockOffset := currPos % blockSize

		if blk, ok := s.blocks[blockIdx]; ok && int(blockOffset) < len(blk) {
			p[i] = blk[blockOffset]
		} else {
			p[i] = 0 // unwritten sectors return zeros
		}
	}

	s.cursor += readLen
	return int(readLen), nil
}

func (s *SimulatedDisk) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.SimulateDisconnect {
		return 0, domain.ErrDeviceRemoved
	}
	if s.InjectedError != nil {
		return 0, s.InjectedError
	}

	if s.FailOnWriteAfterBytes > 0 && s.bytesWritten >= s.FailOnWriteAfterBytes {
		return 0, fmt.Errorf("simulated hardware write failure")
	}

	writeLen := len(p)
	if s.SimulateShortWrite && writeLen > 1024 {
		writeLen = writeLen / 2
	}

	for i := 0; i < writeLen; i++ {
		currPos := s.cursor + int64(i)
		blockIdx := currPos / blockSize
		blockOffset := currPos % blockSize

		blk, ok := s.blocks[blockIdx]
		if !ok {
			blk = make([]byte, blockSize)
			s.blocks[blockIdx] = blk
		}
		blk[blockOffset] = p[i]
	}

	s.cursor += int64(writeLen)
	s.bytesWritten += int64(writeLen)

	if writeLen < len(p) {
		return writeLen, io.ErrShortWrite
	}
	return writeLen, nil
}

func (s *SimulatedDisk) ReadAt(p []byte, off int64) (n int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed || s.SimulateDisconnect {
		return 0, domain.ErrDeviceRemoved
	}
	if off >= s.capacity {
		return 0, io.EOF
	}

	readLen := int64(len(p))
	if off+readLen > s.capacity {
		readLen = s.capacity - off
	}

	for i := int64(0); i < readLen; i++ {
		currPos := off + i
		blockIdx := currPos / blockSize
		blockOffset := currPos % blockSize

		if blk, ok := s.blocks[blockIdx]; ok && int(blockOffset) < len(blk) {
			p[i] = blk[blockOffset]
		} else {
			p[i] = 0
		}
	}

	return int(readLen), nil
}

func (s *SimulatedDisk) WriteAt(p []byte, off int64) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.SimulateDisconnect {
		return 0, domain.ErrDeviceRemoved
	}
	if s.InjectedError != nil {
		return 0, s.InjectedError
	}

	for i := 0; i < len(p); i++ {
		currPos := off + int64(i)
		blockIdx := currPos / blockSize
		blockOffset := currPos % blockSize

		blk, ok := s.blocks[blockIdx]
		if !ok {
			blk = make([]byte, blockSize)
			s.blocks[blockIdx] = blk
		}
		blk[blockOffset] = p[i]
	}

	s.bytesWritten += int64(len(p))
	return len(p), nil
}

func (s *SimulatedDisk) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newCursor int64
	switch whence {
	case io.SeekStart:
		newCursor = offset
	case io.SeekCurrent:
		newCursor = s.cursor + offset
	case io.SeekEnd:
		newCursor = s.capacity + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newCursor < 0 {
		return 0, fmt.Errorf("negative seek position")
	}
	s.cursor = newCursor
	return s.cursor, nil
}

func (s *SimulatedDisk) Sync() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.SimulateDisconnect {
		return domain.ErrDeviceRemoved
	}
	return nil
}

func (s *SimulatedDisk) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *SimulatedDisk) BytesWritten() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bytesWritten
}

// Provider implements device.DeviceProvider for tests.
type Provider struct {
	mu      sync.RWMutex
	devices map[string]domain.BlockDevice
	disks   map[string]*SimulatedDisk
}

func NewFakeProvider() *Provider {
	return &Provider{
		devices: make(map[string]domain.BlockDevice),
		disks:   make(map[string]*SimulatedDisk),
	}
}

func (f *Provider) AddDevice(dev domain.BlockDevice, disk *SimulatedDisk) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.devices[dev.Path] = dev
	if disk != nil {
		f.disks[dev.Path] = disk
	}
}

func (f *Provider) RemoveDevice(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.devices, path)
	if disk, ok := f.disks[path]; ok {
		_ = disk.Close()
		delete(f.disks, path)
	}
}

func (f *Provider) GetDisk(path string) (*SimulatedDisk, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	d, ok := f.disks[path]
	return d, ok
}

func (f *Provider) ListDevices(ctx context.Context) ([]domain.BlockDevice, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var list []domain.BlockDevice
	for _, dev := range f.devices {
		list = append(list, dev)
	}
	return list, nil
}

func (f *Provider) GetDevice(ctx context.Context, path string) (domain.BlockDevice, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	dev, ok := f.devices[path]
	if !ok {
		return domain.BlockDevice{}, fmt.Errorf("%w: %s", domain.ErrDeviceRemoved, path)
	}
	return dev, nil
}

func (f *Provider) GetRawPath(devicePath string) string {
	return devicePath
}
