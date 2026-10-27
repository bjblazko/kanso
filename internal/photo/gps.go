// SPDX-License-Identifier: Apache-2.0

// Package photo removes the location from photos before they are stored, so
// a photo shared in a page does not tell where it was taken.
package photo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"path/filepath"
	"strings"
)

// ErrTooLarge means the photo exceeded the size limit.
var ErrTooLarge = errors.New("photo too large")

const (
	tagGPSInfo = 0x8825
	entrySize  = 12
)

// Clean returns r with the GPS data removed when name is a JPEG; other files
// pass through unread. A JPEG is read into memory, at most max bytes.
func Clean(name string, r io.Reader, max int64) (io.Reader, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".jpg" && ext != ".jpeg" {
		return r, nil
	}
	b, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, ErrTooLarge
	}
	return bytes.NewReader(StripGPS(b)), nil
}

// StripGPS removes the GPS data from a JPEG's EXIF block in place, keeping
// every other tag (such as the orientation) and the file's length. Anything
// that is not a well-formed JPEG with GPS data is returned unchanged.
func StripGPS(b []byte) []byte {
	out := bytes.Clone(b)
	if tiff := exifTIFF(out); tiff != nil && stripTIFF(tiff) {
		return out
	}
	return b
}

// exifTIFF returns the TIFF data of the first EXIF segment, or nil.
func exifTIFF(b []byte) []byte {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return nil
	}
	for pos := 2; pos+4 <= len(b); {
		marker := b[pos+1]
		if b[pos] != 0xFF || marker == 0xDA || marker == 0xD9 {
			return nil
		}
		end := pos + 2 + int(binary.BigEndian.Uint16(b[pos+2:]))
		if end < pos+4 || end > len(b) {
			return nil
		}
		seg := b[pos+4 : end]
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return seg[6:]
		}
		pos = end
	}
	return nil
}

// tiffData is the TIFF block of an EXIF segment with its byte order.
type tiffData struct {
	b []byte
	o binary.ByteOrder
}

func stripTIFF(b []byte) bool {
	t, ok := parseTIFF(b)
	if !ok {
		return false
	}
	ifd0 := uint64(t.o.Uint32(b[4:]))
	n, ok := t.entries(ifd0)
	if !ok {
		return false
	}
	for i := range n {
		e := ifd0 + 2 + entrySize*i
		if t.o.Uint16(b[e:]) != tagGPSInfo {
			continue
		}
		if !t.zeroIFD(uint64(t.o.Uint32(b[e+8:]))) {
			return false
		}
		t.removeEntry(ifd0, n, i)
		return true
	}
	return false
}

func parseTIFF(b []byte) (tiffData, bool) {
	if len(b) < 8 {
		return tiffData{}, false
	}
	var o binary.ByteOrder
	switch string(b[:2]) {
	case "II":
		o = binary.LittleEndian
	case "MM":
		o = binary.BigEndian
	default:
		return tiffData{}, false
	}
	return tiffData{b: b, o: o}, o.Uint16(b[2:]) == 42
}

// entries returns the entry count of the IFD at off if it lies inside.
func (t tiffData) entries(off uint64) (uint64, bool) {
	if off+2 > uint64(len(t.b)) {
		return 0, false
	}
	n := uint64(t.o.Uint16(t.b[off:]))
	return n, off+2+entrySize*n+4 <= uint64(len(t.b))
}

// zeroIFD overwrites the IFD at off and every value it points to.
func (t tiffData) zeroIFD(off uint64) bool {
	n, ok := t.entries(off)
	if !ok {
		return false
	}
	for i := range n {
		e := off + 2 + entrySize*i
		size, known := valueSize(t.o.Uint16(t.b[e+2:]), uint64(t.o.Uint32(t.b[e+4:])))
		if !known {
			return false
		}
		if size > 4 {
			at := uint64(t.o.Uint32(t.b[e+8:]))
			if at+size > uint64(len(t.b)) {
				return false
			}
			clear(t.b[at : at+size])
		}
	}
	clear(t.b[off : off+2+entrySize*n+4])
	return true
}

// removeEntry drops entry i of the IFD at off (n entries), moving the
// following entries and the next-IFD offset up and clearing the freed bytes.
func (t tiffData) removeEntry(off, n, i uint64) {
	start := off + 2 + entrySize*i
	end := off + 2 + entrySize*n + 4
	copy(t.b[start:], t.b[start+entrySize:end])
	clear(t.b[end-entrySize : end])
	t.o.PutUint16(t.b[off:], uint16(n-1))
}

// valueSize is the byte size of count values of a TIFF field type.
func valueSize(typ uint16, count uint64) (uint64, bool) {
	sizes := map[uint16]uint64{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8, 11: 4, 12: 8}
	s, ok := sizes[typ]
	return s * count, ok && count <= 1<<28
}
