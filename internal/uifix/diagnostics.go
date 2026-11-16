package uifix

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const diagnosticCapacity = 64
const diagnosticEntryBytes = 512
const diagnosticMappingBytes = 256 + 16 + diagnosticCapacity*diagnosticEntryBytes

type diagnosticCursor struct{ session, sequence uint64 }
type FallbackRecord struct {
	ExcludedFromStats bool
	Sequence          uint64
	At                time.Time
	Reason            string
	Interface         string
	Caller            uint32
}

func fallbackReason(reason uint32) string {
	switch reason {
	case 1:
		return "快照超过3秒未更新"
	case 2:
		return "接口名称参数无效"
	case 3:
		return "接口名称或索引未命中快照"
	case 4:
		return "缓存查询发生异常"
	case 5:
		return "空接口名称，已直接返回无效接口，不计入命中率统计"
	default:
		return fmt.Sprintf("未知原因(%d)", reason)
	}
}

func (m *Manager) DrainFallbacks(pid uint32) ([]FallbackRecord, uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf("Local\\EngineToolsCache-%d", pid))
	raw, _, _ := kernel.NewProc("OpenFileMappingW").Call(windows.FILE_MAP_READ, 0, uintptr(unsafe.Pointer(name)))
	if raw == 0 {
		return nil, 0
	}
	handle := windows.Handle(raw)
	defer windows.CloseHandle(handle)
	address, err := windows.MapViewOfFile(handle, windows.FILE_MAP_READ, 0, 0, diagnosticMappingBytes)
	if err != nil {
		return nil, 0
	} // Older DLLs expose statistics only.
	defer windows.UnmapViewOfFile(address)
	return m.drainFallbackMapping(pid, address)
}

func (m *Manager) drainFallbackMapping(pid uint32, address uintptr) ([]FallbackRecord, uint64) {
	load := func(offset uintptr) uint64 { return atomic.LoadUint64((*uint64)(unsafe.Pointer(address + offset))) }
	if load(0) != 0x45435431 || load(8) != cacheProtocol || load(16) != uint64(pid) {
		return nil, 0
	}
	if m.diagnostics == nil {
		m.diagnostics = make(map[uint32]diagnosticCursor)
	}
	count, session := load(256), load(264)
	cursor := m.diagnostics[pid]
	if cursor.session != session || cursor.sequence > count {
		cursor = diagnosticCursor{session: session}
	}
	first := uint64(1)
	if count >= diagnosticCapacity {
		first = count - diagnosticCapacity + 1
	}
	var dropped uint64
	if cursor.sequence+1 < first {
		dropped = first - cursor.sequence - 1
		cursor.sequence = first - 1
	}
	var records []FallbackRecord
	for sequence := cursor.sequence + 1; sequence <= count; sequence++ {
		offset := uintptr(272 + ((sequence-1)%diagnosticCapacity)*diagnosticEntryBytes)
		if load(offset) != sequence {
			break
		}
		stamp := load(offset + 8)
		reason := *(*uint32)(unsafe.Pointer(address + offset + 16))
		caller := *(*uint32)(unsafe.Pointer(address + offset + 20))
		var text [240]uint16
		copy(text[:], (*[240]uint16)(unsafe.Pointer(address + offset + 24))[:])
		if load(offset) != sequence {
			break
		}
		ft := windows.Filetime{LowDateTime: uint32(stamp), HighDateTime: uint32(stamp >> 32)}
		records = append(records, FallbackRecord{ExcludedFromStats: reason == 5, Sequence: sequence, At: time.Unix(0, ft.Nanoseconds()), Reason: fallbackReason(reason), Interface: windows.UTF16ToString(text[:]), Caller: caller})
		cursor.sequence = sequence
	}
	// Only one selected Engine DJ process is monitored at a time.
	clear(m.diagnostics)
	m.diagnostics[pid] = cursor
	return records, dropped
}
