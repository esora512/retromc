package mcregion

import (
	"encoding/binary"
	"fmt"
	"math"
)

type TagType byte

const (
	TagEnd       TagType = 0
	TagByte      TagType = 1
	TagShort     TagType = 2
	TagInt       TagType = 3
	TagLong      TagType = 4
	TagByteArray TagType = 7
	TagString    TagType = 8
	TagList      TagType = 9
	TagCompound  TagType = 10
	TagFloat     TagType = 5
	TagDouble    TagType = 6
)

// Tag is a parsed NBT tag. Only the fields matching Type are meaningful.
type Tag struct {
	Type      TagType
	Name      string
	ByteVal   byte
	ShortVal  int16
	IntVal    int32
	LongVal   int64
	StrVal    string
	ByteArr   []byte
	ListType  TagType
	List      []*Tag
	Compound  map[string]*Tag
	FloatVal  float32
	DoubleVal float64
}

// Get returns the named child of a compound tag, or nil if absent /
// t isn't a compound. Safe to call on a nil receiver.
func (t *Tag) Get(name string) *Tag {
	if t == nil || t.Compound == nil {
		return nil
	}
	return t.Compound[name]
}

func (t *Tag) Has(name string) bool { return t.Get(name) != nil }

type nbtReader struct {
	buf []byte
	pos int
}

func (r *nbtReader) u8() (byte, error) {
	if r.pos+1 > len(r.buf) {
		return 0, fmt.Errorf("nbt: EOF reading byte")
	}
	v := r.buf[r.pos]
	r.pos++
	return v, nil
}

func (r *nbtReader) i16() (int16, error) {
	if r.pos+2 > len(r.buf) {
		return 0, fmt.Errorf("nbt: EOF reading short")
	}
	v := int16(binary.BigEndian.Uint16(r.buf[r.pos:]))
	r.pos += 2
	return v, nil
}

func (r *nbtReader) i32() (int32, error) {
	if r.pos+4 > len(r.buf) {
		return 0, fmt.Errorf("nbt: EOF reading int")
	}
	v := int32(binary.BigEndian.Uint32(r.buf[r.pos:]))
	r.pos += 4
	return v, nil
}

func (r *nbtReader) i64() (int64, error) {
	if r.pos+8 > len(r.buf) {
		return 0, fmt.Errorf("nbt: EOF reading long")
	}
	v := int64(binary.BigEndian.Uint64(r.buf[r.pos:]))
	r.pos += 8
	return v, nil
}

func (r *nbtReader) bytes(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.buf) {
		return nil, fmt.Errorf("nbt: EOF reading %d bytes", n)
	}
	v := r.buf[r.pos : r.pos+n]
	r.pos += n
	return v, nil
}

func (r *nbtReader) name() (string, error) {
	n, err := r.i16()
	if err != nil {
		return "", err
	}
	b, err := r.bytes(int(n))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *nbtReader) readPayload(t TagType) (*Tag, error) {
	tag := &Tag{Type: t}
	switch t {
	case TagByte:
		v, err := r.u8()
		if err != nil {
			return nil, err
		}
		tag.ByteVal = v
	case TagShort:
		v, err := r.i16()
		if err != nil {
			return nil, err
		}
		tag.ShortVal = v
	case TagInt:
		v, err := r.i32()
		if err != nil {
			return nil, err
		}
		tag.IntVal = v
	case TagLong:
		v, err := r.i64()
		if err != nil {
			return nil, err
		}
		tag.LongVal = v
	case TagByteArray:
		n, err := r.i32()
		if err != nil {
			return nil, err
		}
		b, err := r.bytes(int(n))
		if err != nil {
			return nil, err
		}
		tag.ByteArr = append([]byte(nil), b...)
	case TagString:
		n, err := r.i16()
		if err != nil {
			return nil, err
		}
		b, err := r.bytes(int(n))
		if err != nil {
			return nil, err
		}
		tag.StrVal = string(b)
	case TagList:
		elemType, err := r.u8()
		if err != nil {
			return nil, err
		}
		count, err := r.i32()
		if err != nil {
			return nil, err
		}
		tag.ListType = TagType(elemType)
		if tag.ListType != TagEnd {
			tag.List = make([]*Tag, 0, count)
			for i := int32(0); i < count; i++ {
				child, err := r.readPayload(tag.ListType)
				if err != nil {
					return nil, err
				}
				tag.List = append(tag.List, child)
			}
		}
	case TagCompound:
		tag.Compound = make(map[string]*Tag)
		for {
			childType, err := r.u8()
			if err != nil {
				return nil, err
			}
			if TagType(childType) == TagEnd {
				break
			}
			childName, err := r.name()
			if err != nil {
				return nil, err
			}
			child, err := r.readPayload(TagType(childType))
			if err != nil {
				return nil, err
			}
			child.Name = childName
			tag.Compound[childName] = child
		}
	case TagFloat:
		v, err := r.i32()
		if err != nil {
			return nil, err
		}
		tag.FloatVal = math.Float32frombits(uint32(v))
	case TagDouble:
		v, err := r.i64()
		if err != nil {
			return nil, err
		}
		tag.DoubleVal = math.Float64frombits(uint64(v))
	default:
		return nil, fmt.Errorf("nbt: unsupported tag type %d", t)
	}
	return tag, nil
}

// ParseRoot parses TAG_Compound(""), contents, TAG_End — the format
// produced by Compound.Root().
func ParseRoot(data []byte) (*Tag, error) {
	r := &nbtReader{buf: data}
	rootType, err := r.u8()
	if err != nil {
		return nil, err
	}
	if TagType(rootType) != TagCompound {
		return nil, fmt.Errorf("nbt: expected root TAG_Compound, got %d", rootType)
	}
	if _, err := r.name(); err != nil {
		return nil, err
	}
	return r.readPayload(TagCompound)
}
