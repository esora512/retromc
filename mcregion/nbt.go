package mcregion

// A tiny NBT (big-endian) encoder — just enough to write Beta 1.7.3 chunk
// and level.dat data. Not a general-purpose NBT library.

import (
	"bytes"
	"encoding/binary"
)

const (
	tagEnd       = 0
	tagByte      = 1
	tagShort     = 2
	tagInt       = 3
	tagLong      = 4
	tagByteArray = 7
	tagString    = 8
	tagList      = 9
	tagCompound  = 10
)

// Compound is an ordered, named NBT compound tag being built up for writing.
type Compound struct {
	buf bytes.Buffer
}

func NewCompound() *Compound {
	return &Compound{}
}

func writeName(buf *bytes.Buffer, name string) {
	binary.Write(buf, binary.BigEndian, uint16(len(name)))
	buf.WriteString(name)
}

func (c *Compound) Byte(name string, v byte) {
	c.buf.WriteByte(tagByte)
	writeName(&c.buf, name)
	c.buf.WriteByte(v)
}

func (c *Compound) Short(name string, v int16) {
	c.buf.WriteByte(tagShort)
	writeName(&c.buf, name)
	binary.Write(&c.buf, binary.BigEndian, v)
}

func (c *Compound) Int(name string, v int32) {
	c.buf.WriteByte(tagInt)
	writeName(&c.buf, name)
	binary.Write(&c.buf, binary.BigEndian, v)
}

func (c *Compound) Long(name string, v int64) {
	c.buf.WriteByte(tagLong)
	writeName(&c.buf, name)
	binary.Write(&c.buf, binary.BigEndian, v)
}

func (c *Compound) String(name, v string) {
	c.buf.WriteByte(tagString)
	writeName(&c.buf, name)
	binary.Write(&c.buf, binary.BigEndian, uint16(len(v)))
	c.buf.WriteString(v)
}

func (c *Compound) ByteArray(name string, v []byte) {
	c.buf.WriteByte(tagByteArray)
	writeName(&c.buf, name)
	binary.Write(&c.buf, binary.BigEndian, int32(len(v)))
	c.buf.Write(v)
}

// EmptyList writes a zero-length list tag (valid regardless of element type
// when count is 0 — used for Entities/TileEntities/TileTicks).
func (c *Compound) EmptyList(name string) {
	c.buf.WriteByte(tagList)
	writeName(&c.buf, name)
	c.buf.WriteByte(tagEnd) // element type
	binary.Write(&c.buf, binary.BigEndian, int32(0))
}

// AddCompound merges a fully-built child compound into this one under the
// given tag name. Build the child with NewCompound() first.
func (c *Compound) AddCompound(name string, child *Compound) {
	c.buf.WriteByte(tagCompound)
	writeName(&c.buf, name)
	c.buf.Write(child.Bytes())
	c.buf.WriteByte(tagEnd)
}

// Bytes returns the encoded tag payload (without the trailing End tag —
// callers append that themselves via AddCompound / Root).
func (c *Compound) Bytes() []byte {
	return c.buf.Bytes()
}

// Root wraps this compound as the unnamed root tag of an NBT file/blob:
// TAG_Compound(""), contents, TAG_End.
func (c *Compound) Root() []byte {
	var out bytes.Buffer
	out.WriteByte(tagCompound)
	writeName(&out, "")
	out.Write(c.buf.Bytes())
	out.WriteByte(tagEnd)
	return out.Bytes()
}