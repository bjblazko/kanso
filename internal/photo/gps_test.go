// SPDX-License-Identifier: Apache-2.0

package photo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// order is a byte order that can also append.
type order interface {
	binary.ByteOrder
	binary.AppendByteOrder
}

const (
	tagOrientation = 0x0112
	tagGPS         = 0x8825
)

// latitude is GPSLatitude 48/1, 8/1, 1234/100 in the given byte order.
func latitude(o order) []byte {
	var b []byte
	for _, v := range []uint32{48, 1, 8, 1, 1234, 100} {
		b = o.AppendUint32(b, v)
	}
	return b
}

func entry(o order, tag, typ uint16, count, value uint32) []byte {
	b := o.AppendUint16(nil, tag)
	b = o.AppendUint16(b, typ)
	b = o.AppendUint32(b, count)
	return o.AppendUint32(b, value)
}

// tiff builds IFD0 {Orientation=6[, GPSInfo → GPS IFD {GPSLatitude}]}.
func tiff(o order, withGPS bool) []byte {
	mark := "II"
	if o.String() == binary.BigEndian.String() {
		mark = "MM"
	}
	t := append([]byte(mark), 0, 0, 0, 0, 0, 0)
	o.PutUint16(t[2:], 42)
	o.PutUint32(t[4:], 8)
	n := uint16(1)
	if withGPS {
		n = 2
	}
	t = o.AppendUint16(t, n)
	orientation := o.AppendUint16(nil, 6)
	t = append(t, entry(o, tagOrientation, 3, 1, 0)...)
	copy(t[len(t)-4:], orientation)
	gpsIFD := uint32(8 + 2 + 12*int(n) + 4)
	if withGPS {
		t = append(t, entry(o, tagGPS, 4, 1, gpsIFD)...)
	}
	t = o.AppendUint32(t, 0)
	if withGPS {
		t = o.AppendUint16(t, 1)
		t = append(t, entry(o, 0x0002, 5, 3, gpsIFD+18)...)
		t = o.AppendUint32(t, 0)
		t = append(t, latitude(o)...)
	}
	return t
}

func jpegWithExif(t []byte) []byte {
	seg := append([]byte("Exif\x00\x00"), t...)
	b := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	b = binary.BigEndian.AppendUint16(b, uint16(len(seg)+2))
	b = append(b, seg...)
	return append(b, 0xFF, 0xDA, 0x00, 0x02, 0x11, 0xFF, 0xD9)
}

// ifd0 reads the tags and SHORT values of IFD0 from a JPEG built above.
func ifd0(t *testing.T, b []byte, o order) map[uint16]uint16 {
	t.Helper()
	tf := b[4+2+6:]
	off := o.Uint32(tf[4:])
	n := o.Uint16(tf[off:])
	tags := map[uint16]uint16{}
	for i := range uint32(n) {
		e := tf[off+2+12*i:]
		tags[o.Uint16(e)] = o.Uint16(e[8:])
	}
	return tags
}

func TestStripGPSRemovesLocationKeepsOrientation(t *testing.T) {
	for _, o := range []order{binary.LittleEndian, binary.BigEndian} {
		in := jpegWithExif(tiff(o, true))
		out := StripGPS(bytes.Clone(in))
		if len(out) != len(in) {
			t.Fatalf("%v: length %d → %d", o, len(in), len(out))
		}
		if bytes.Contains(out, latitude(o)) {
			t.Errorf("%v: latitude still in the file", o)
		}
		tags := ifd0(t, out, o)
		if _, ok := tags[tagGPS]; ok {
			t.Errorf("%v: GPS pointer still in IFD0", o)
		}
		if tags[tagOrientation] != 6 {
			t.Errorf("%v: orientation = %d", o, tags[tagOrientation])
		}
	}
}

func TestStripGPSLeavesOtherFilesAlone(t *testing.T) {
	withGPS := jpegWithExif(tiff(binary.LittleEndian, true))
	inputs := [][]byte{
		[]byte("\x89PNG\r\n\x1a\n rest"),
		{0xFF, 0xD8, 0xFF, 0xDA, 0x00, 0x02, 0xFF, 0xD9},
		jpegWithExif(tiff(binary.LittleEndian, false)),
		withGPS[:3], withGPS[:20], withGPS[:40],
		[]byte("random bytes that are no image"),
		{},
	}
	for i, in := range inputs {
		if out := StripGPS(bytes.Clone(in)); !bytes.Equal(out, in) {
			t.Errorf("input %d changed", i)
		}
	}
}

func TestStripGPSRejectsOffsetsOutsideTheSegment(t *testing.T) {
	tf := tiff(binary.LittleEndian, true)
	binary.LittleEndian.PutUint32(tf[8+2+12+8:], 0xFFFFFF00) // GPS IFD far outside
	in := jpegWithExif(tf)
	if out := StripGPS(bytes.Clone(in)); !bytes.Equal(out, in) {
		t.Error("broken file was changed")
	}
}

func TestCleanOnlyTouchesJPEGNames(t *testing.T) {
	in := jpegWithExif(tiff(binary.LittleEndian, true))
	r, err := Clean("x.PNG", bytes.NewReader(in), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := io.ReadAll(r); !bytes.Equal(got, in) {
		t.Error("PNG name was changed")
	}
	r, err = Clean("x.JPG", bytes.NewReader(in), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := io.ReadAll(r); bytes.Contains(got, latitude(binary.LittleEndian)) {
		t.Error("JPG keeps GPS")
	}
	if _, err := Clean("x.jpeg", strings.NewReader("123456"), 5); !errors.Is(err, ErrTooLarge) {
		t.Errorf("too large: %v", err)
	}
}

func TestFixtureMatchesBuilder(t *testing.T) {
	b, err := os.ReadFile("testdata/gps.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, jpegWithExif(tiff(binary.LittleEndian, true))) {
		t.Error("testdata/gps.jpg differs from the builder")
	}
}
