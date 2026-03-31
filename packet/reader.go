package packet

import (
	"encoding/binary"
	"io"
	"math"
	"unicode/utf16"
)

const (
	BYTE_SIZE   = 1
	SHORT_SIZE  = 2
	INT_SIZE    = 4
	LONG_SIZE   = 8
	FLOAT_SIZE  = 4
	DOUBLE_SIZE = 8
	BOOL_SIZE   = 1

	STRING_CHARACTER_SIZE = 2
)

type PacketReader struct {
	Reader   io.Reader
	PacketId byte
}

func NewReader(r io.Reader, packetId byte) *PacketReader {
	return &PacketReader{Reader: r, PacketId: packetId}
}

func (r *PacketReader) readFull(buf []byte) {
	_, err := io.ReadFull(r.Reader, buf)
	if err != nil {
		panic(err)
	}
}

func (r *PacketReader) ReadByte() byte {
	buf := make([]byte, BYTE_SIZE)
	r.readFull(buf)
	return buf[0]
}

func (r *PacketReader) GetPacketId() byte {
	return r.PacketId
}

func (r *PacketReader) ReadBool() bool {
	// 0x00 = False; 0x01 = True
	return r.ReadByte() == 0x01
}

func (r *PacketReader) ReadShort() uint16 {
	data := make([]byte, SHORT_SIZE)
	r.readFull(data)
	return binary.BigEndian.Uint16(data)
}

func (r *PacketReader) ReadInt32() int32 {
	data := make([]byte, INT_SIZE)
	r.readFull(data)
	return int32(binary.BigEndian.Uint32(data))
}

func (r *PacketReader) ReadInt() int {
	data := make([]byte, INT_SIZE)
	r.readFull(data)
	return int(binary.BigEndian.Uint32(data))
}

func (r *PacketReader) ReadLong() int64 {
	data := make([]byte, LONG_SIZE)
	r.readFull(data)
	return int64(binary.BigEndian.Uint64(data))
}

func (r *PacketReader) ReadFloat32() float32 {
	data := make([]byte, FLOAT_SIZE)
	r.readFull(data)
	bits := binary.BigEndian.Uint32(data)
	return math.Float32frombits(bits)
}

func (r *PacketReader) ReadFloat64() float64 {
	data := make([]byte, DOUBLE_SIZE)
	r.readFull(data)
	bits := binary.BigEndian.Uint64(data)
	return math.Float64frombits(bits)
}

func (r *PacketReader) ReadString16() string {
	strLength := r.ReadShort()
	strLengthBytes := int(strLength) * STRING_CHARACTER_SIZE
	strData := make([]byte, strLengthBytes)
	r.readFull(strData)
	return string(strData)
}

func (r *PacketReader) Read (b []byte) {
	r.readFull(b)
}

func (r *PacketReader) ReadString16AndDecodeUTF16() string {
	strLength := r.ReadShort()

	// convert string length to bytes
	strLengthBytes := int(strLength) * STRING_CHARACTER_SIZE

	// read string bytes from reader
	strData := make([]byte, strLengthBytes)
	r.readFull(strData)

	return decodeUTF16(strData)
}

func decodeUTF16(b []byte) string {
	// Convert bytes to uint16 slice
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.BigEndian.Uint16(b[i*2 : i*2+2])
	}
	return string(utf16.Decode(u16))
}
