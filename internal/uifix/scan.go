package uifix

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const interfaceLookupSymbol = "?interfaceFromName@QNetworkInterface@@SA?AV1@AEBVQString@@@Z"

var requiredQtExports = map[string][]string{
	"Qt6Core.dll": {"??1QString@@QEAA@XZ", "?deallocate@QArrayData@@SAXPEAU1@_J1@Z", "qVersion"},
	"Qt6Network.dll": {
		interfaceLookupSymbol,
		"??0QNetworkInterface@@QEAA@XZ", "??0QNetworkInterface@@QEAA@AEBV0@@Z", "??1QNetworkInterface@@QEAA@XZ",
		"?name@QNetworkInterface@@QEBA?AVQString@@XZ", "?index@QNetworkInterface@@QEBAHXZ",
		"?allInterfaces@QNetworkInterface@@SA?AV?$QList@VQNetworkInterface@@@@XZ",
	},
}

type imageReader struct {
	file   *pe.File
	data   []byte
	header *pe.OptionalHeader64
}

func readImage(path, expectedHash string) (*imageReader, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 64 || info.Size() > 256<<20 {
		return nil, fmt.Errorf("unsupported PE file size: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if expectedHash != "" && hex.EncodeToString(digest[:]) != expectedHash {
		return nil, fmt.Errorf("file changed during compatibility scan: %s", path)
	}
	file, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid PE image %s: %w", path, err)
	}
	header, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if file.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || !ok {
		return nil, fmt.Errorf("automatic cache detection requires Windows x64: %s", path)
	}
	return &imageReader{file: file, data: data, header: header}, nil
}
func (r *imageReader) read(rva uint32, size uint64) ([]byte, error) {
	for _, section := range r.file.Sections {
		if uint64(rva) >= uint64(section.VirtualAddress) && uint64(rva)+size <= uint64(section.VirtualAddress)+uint64(section.Size) {
			start := uint64(section.Offset) + uint64(rva-section.VirtualAddress)
			if start+size <= uint64(len(r.data)) {
				return r.data[start : start+size], nil
			}
		}
	}
	return nil, fmt.Errorf("PE RVA 0x%x is outside file-backed sections", rva)
}
func (r *imageReader) text(rva uint32) (string, error) {
	if rva == 0 {
		return "", fmt.Errorf("null PE name RVA")
	}
	var name []byte
	for i := uint32(0); i < 4096 && uint64(rva)+uint64(i) <= 0xffffffff; i++ {
		b, err := r.read(rva+i, 1)
		if err != nil {
			return "", err
		}
		if b[0] == 0 {
			return string(name), nil
		}
		name = append(name, b[0])
	}
	return "", fmt.Errorf("unterminated PE name")
}
func (r *imageReader) exports() (map[string]bool, error) {
	dir := r.header.DataDirectory[0]
	if dir.VirtualAddress == 0 || dir.Size < 40 {
		return nil, fmt.Errorf("missing PE exports")
	}
	table, err := r.read(dir.VirtualAddress, 40)
	if err != nil {
		return nil, err
	}
	count := binary.LittleEndian.Uint32(table[24:28])
	names := binary.LittleEndian.Uint32(table[32:36])
	if count > 65536 {
		return nil, fmt.Errorf("too many PE exports")
	}
	array, err := r.read(names, uint64(count)*4)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, count)
	for i := uint32(0); i < count; i++ {
		name, err := r.text(binary.LittleEndian.Uint32(array[i*4:]))
		if err != nil {
			return nil, err
		}
		result[name] = true
	}
	return result, nil
}
func (r *imageReader) lookupSlot() (uint32, error) {
	dir := r.header.DataDirectory[1]
	if dir.VirtualAddress == 0 || dir.Size < 20 || dir.Size > 1<<20 {
		return 0, fmt.Errorf("missing/invalid import directory")
	}
	var slot uint32
	terminated := false
	for offset := uint32(0); offset+20 <= dir.Size; offset += 20 {
		descriptor, err := r.read(dir.VirtualAddress+offset, 20)
		if err != nil {
			return 0, err
		}
		if bytes.Equal(descriptor, make([]byte, 20)) {
			terminated = true
			break
		}
		name, err := r.text(binary.LittleEndian.Uint32(descriptor[12:16]))
		if err != nil {
			return 0, err
		}
		if !strings.EqualFold(name, "Qt6Network.dll") {
			continue
		}
		original := binary.LittleEndian.Uint32(descriptor[0:4])
		iat := binary.LittleEndian.Uint32(descriptor[16:20])
		if original == 0 || iat == 0 {
			return 0, fmt.Errorf("Qt imports have no on-disk lookup table")
		}
		ended := false
		for index := uint32(0); index < 16384; index++ {
			thunk, err := r.read(original+index*8, 8)
			if err != nil {
				return 0, err
			}
			address := binary.LittleEndian.Uint64(thunk)
			if address == 0 {
				ended = true
				break
			}
			if address>>63 != 0 {
				continue
			}
			if address > 0xfffffffd {
				return 0, fmt.Errorf("invalid import name RVA")
			}
			symbol, err := r.text(uint32(address) + 2)
			if err != nil {
				return 0, err
			}
			if symbol == interfaceLookupSymbol {
				if slot != 0 {
					return 0, fmt.Errorf("ambiguous Qt interface lookup import")
				}
				slot = iat + index*8
				if _, err := r.read(slot, 8); err != nil {
					return 0, err
				}
			}
		}
		if !ended {
			return 0, fmt.Errorf("unterminated import lookup table")
		}
	}
	if !terminated || slot == 0 {
		return 0, fmt.Errorf("no supported Qt interface lookup import found")
	}
	return slot, nil
}

type functionRange struct{ start, end uint32 }

func (r *imageReader) functions() ([]functionRange, error) {
	dir := r.header.DataDirectory[3]
	if dir.VirtualAddress == 0 || dir.Size == 0 || dir.Size%12 != 0 || dir.Size > 24<<20 {
		return nil, fmt.Errorf("missing/invalid x64 exception directory")
	}
	data, err := r.read(dir.VirtualAddress, uint64(dir.Size))
	if err != nil {
		return nil, err
	}
	result := make([]functionRange, 0, len(data)/12)
	for offset := 0; offset < len(data); offset += 12 {
		start := binary.LittleEndian.Uint32(data[offset:])
		end := binary.LittleEndian.Uint32(data[offset+4:])
		if start >= end || end > r.header.SizeOfImage {
			return nil, fmt.Errorf("invalid x64 function range")
		}
		result = append(result, functionRange{start, end})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].start < result[j].start })
	return result, nil
}
func (r *imageReader) scanCallers(slot uint32) ([]uint32, error) {
	functions, err := r.functions()
	if err != nil {
		return nil, err
	}
	var callers []uint32
	for _, section := range r.file.Sections {
		if section.Characteristics&pe.IMAGE_SCN_MEM_EXECUTE == 0 {
			continue
		}
		size := section.Size
		if section.VirtualSize != 0 && section.VirtualSize < size {
			size = section.VirtualSize
		}
		code, err := r.read(section.VirtualAddress, uint64(size))
		if err != nil {
			return nil, err
		}
		for offset := 0; offset+6 <= len(code); offset++ {
			if code[offset] != 0xff || code[offset+1] != 0x15 {
				continue
			}
			call := uint64(section.VirtualAddress) + uint64(offset)
			target := int64(call+6) + int64(int32(binary.LittleEndian.Uint32(code[offset+2:])))
			if target != int64(slot) {
				continue
			}
			index := sort.Search(len(functions), func(i int) bool { return uint64(functions[i].start) > call }) - 1
			if index < 0 || uint64(functions[index].end) < call+6 {
				return nil, fmt.Errorf("query call outside a recognized x64 function")
			}
			callers = append(callers, uint32(call+6))
			if len(callers) > maxCacheCallers {
				return nil, fmt.Errorf("more than %d query call sites; automatic cache detection is not supported", maxCacheCallers)
			}
		}
	}
	if len(callers) == 0 {
		return nil, fmt.Errorf("no supported direct Qt query calls found")
	}
	sort.Slice(callers, func(i, j int) bool { return callers[i] < callers[j] })
	return callers, nil
}

func scanAutomaticBuild(directory string, hashes map[string]string) (*buildProfile, error) {
	for name, required := range requiredQtExports {
		image, err := readImage(filepath.Join(directory, name), hashes[name])
		if err != nil {
			return nil, err
		}
		exports, err := image.exports()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for _, symbol := range required {
			if !exports[symbol] {
				return nil, fmt.Errorf("%s lacks required Qt interface %s", name, symbol)
			}
		}
	}
	image, err := readImage(filepath.Join(directory, "Engine DJ.exe"), hashes["Engine DJ.exe"])
	if err != nil {
		return nil, err
	}
	slot, err := image.lookupSlot()
	if err != nil {
		return nil, err
	}
	callers, err := image.scanCallers(slot)
	if err != nil {
		return nil, err
	}
	return &buildProfile{Name: "Automatic Qt capability detection", Hashes: hashes, Callers: callers, Automatic: true}, nil
}
