package metadata

import (
	"bytes"
	"os"
)

// DetectTagVersion infers the tag family backing an audio file directly
// from its bytes, mirroring what TagLib exposes per container:
//
//	mp3 -> "id3v22" / "id3v23" / "id3v24" / "id3v1" / "none"
//	flac -> "vorbis"
//	wav/aiff -> "id3v23" when an ID3 chunk exists, else "none"
func DetectTagVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 16 {
		return "none"
	}
	switch {
	case isFLAC(data):
		return "vorbis"
	case hasMPEGSync(data):
		return mpegTagVersion(data)
	default:
		if bytes.Contains(data, []byte("ID3 ")) {
			return "id3v23"
		}
		return "none"
	}
}

func isFLAC(data []byte) bool { return bytes.HasPrefix(data, []byte("fLaC")) }

func hasMPEGSync(data []byte) bool {
	// skip an existing ID3v2 header if present
	off := 0
	if bytes.HasPrefix(data, []byte("ID3")) && len(data) >= 10 {
		size := int(data[6]&0x7f)<<21 | int(data[7]&0x7f)<<14 |
			int(data[8]&0x7f)<<7 | int(data[9]&0x7f)
		off = 10 + size
	}
	return len(data) > off+1 && data[off] == 0xFF && data[off+1]&0xE0 == 0xE0
}

func mpegTagVersion(data []byte) string {
	if bytes.HasPrefix(data, []byte("ID3")) && len(data) >= 10 {
		major := int(data[3] - '0')
		if major < 2 || major > 4 {
			major = 3
		}
		return "id3v2" + itoa(major)
	}
	tail := data
	if len(tail) > 128 {
		tail = tail[len(tail)-128:]
	}
	if bytes.HasPrefix(tail, []byte("TAG")) {
		return "id3v1"
	}
	return "none"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
