package packet

import (
	"bufio"
	"io"
	"log"
	"math"
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
	PacketId byte
	Reader   *bufio.Reader
}

func NewReader(reader *bufio.Reader) PacketReader {
	return PacketReader{
		Reader: reader,
	}
}

func (r *PacketReader) ReadByte() byte {
	b, err := r.Reader.ReadByte()
	if err != nil {
		log.Println("Failed to read byte:", err.Error())
		return 0
	}
	return b
}

func (r *PacketReader) GetPacketId() byte {
	return r.PacketId
}

func (r *PacketReader) ReadBool() bool {
	return r.ReadByte() != 0
}

func (r *PacketReader) ReadShort() uint16 {
	var buf [2]byte
	io.ReadFull(r.Reader, buf[:])
	return uint16(buf[0])<<8 | uint16(buf[1])
}

func (r *PacketReader) ReadInt32() int32 {
	var buf [4]byte
	io.ReadFull(r.Reader, buf[:])
	return int32(buf[0])<<24 | int32(buf[1])<<16 | int32(buf[2])<<8 | int32(buf[3])
}

func (r *PacketReader) ReadInt() int {
	return int(r.ReadInt32())
}

func (r *PacketReader) ReadLong() int64 {
	var buf [8]byte
	io.ReadFull(r.Reader, buf[:])
	return int64(buf[0])<<56 | int64(buf[1])<<48 | int64(buf[2])<<40 | int64(buf[3])<<32 |
		int64(buf[4])<<24 | int64(buf[5])<<16 | int64(buf[6])<<8 | int64(buf[7])
}

func (r *PacketReader) ReadFloat32() float32 {
	bits := uint32(r.ReadInt32())
	return math.Float32frombits(bits)
}

func (r *PacketReader) ReadFloat64() float64 {
	bits := uint64(r.ReadLong())
	return math.Float64frombits(bits)
}

func (r *PacketReader) ReadString16() string {
	length := r.ReadShort()
	buf := make([]byte, length*2)
	io.ReadFull(r.Reader, buf)
	// decode UTF-16 BE
	runes := make([]rune, length)
	for i := range runes {
		runes[i] = rune(uint16(buf[i*2])<<8 | uint16(buf[i*2+1]))
	}
	return string(runes)
}
