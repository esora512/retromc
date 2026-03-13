package packet

import (
	"bytes"
	"encoding/binary"
	"math"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type PacketWriter struct {
	buf bytes.Buffer
}

func NewPacketWriter() PacketWriter {
	return PacketWriter{}
}

func (w *PacketWriter) Write(b []byte) {
	w.buf.Write(b)
}

func (w *PacketWriter) WriteByte(b byte) {
	w.Write([]byte{b})
}

func (w *PacketWriter) WriteBool(value bool) {
	if value {
		w.WriteByte(0x01)
	} else {
		w.WriteByte(0x00)
	}
}

func (w *PacketWriter) WriteShort(value uint16) {
	data := make([]byte, SHORT_SIZE)
	binary.BigEndian.PutUint16(data, value)
	w.Write(data)
}

func (w *PacketWriter) WriteInt16(value int16) {
	data := make([]byte, SHORT_SIZE)
	binary.BigEndian.PutUint16(data, uint16(value))
	w.Write(data)
}

func (w *PacketWriter) WriteInt32(value int32) {
	data := make([]byte, INT_SIZE)
	binary.BigEndian.PutUint32(data, uint32(value))
	w.Write(data)
}

func (w *PacketWriter) WriteInt64(value int64) {
	data := make([]byte, LONG_SIZE)
	binary.BigEndian.PutUint64(data, uint64(value))
	w.Write(data)
}

func (w *PacketWriter) WriteFloat32(value float32) {
	bits := math.Float32bits(value)
	data := make([]byte, FLOAT_SIZE)
	binary.BigEndian.PutUint32(data, bits)
	w.Write(data)
}

func (w *PacketWriter) WriteFloat64(value float64) {
	bits := math.Float64bits(value)
	data := make([]byte, DOUBLE_SIZE)
	binary.BigEndian.PutUint64(data, bits)
	w.Write(data)
}

func (w *PacketWriter) WriteString16(s string) {
	encoded, _, _ := transform.Bytes(unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder(), []byte(s))
	w.WriteShort(uint16(len(s)))
	w.Write(encoded)
}

func (w *PacketWriter) WriteString8(s string) {
	w.WriteShort(uint16(len(s)))
	w.Write([]byte(s))
}

func (w *PacketWriter) Bytes() []byte {
	return w.buf.Bytes()
}
