package level

import (
	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/packet"
)

const (
	CROP_MAX_STATE = 7
)

type Growable interface {
	AdvanceState(w *World, bk *BlockKey)
	GrowNow(w *World) bool
}

type Crops struct {
	StartTick int64
	State     byte
}

// TODO: Remove duplicate somehow later -_-
type BlockChangeOutPacket struct {
	X         int32
	Y         byte
	Z         int32
	BlockType byte
	BlockMeta byte
}

func (w *World) SetGrowable(block Block, bk BlockKey) {
	if block.TypeId == byte(constants.Wheat.Value) {
		w.Growables[bk] = &Crops{StartTick: w.Tick, State: block.Metadata}
	}
}

func (w *World) GrowPhysics() {
	for key, growable := range w.Growables {
		if growable.GrowNow(w) {
			growable.AdvanceState(w, &key)
		}
	}
}

func (p *BlockChangeOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.BlockChange)
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteByte(p.BlockType)
	writer.WriteByte(p.BlockMeta)
	return writer.Bytes()
}

func (c *Crops) GrowNow(w *World) bool {
	diff := w.Tick - c.StartTick
	if diff < 0 {
		c.StartTick = w.Tick
		return false
	}
	if diff > 600 {
		return true
	}
	return false
}

func (c *Crops) AdvanceState(w *World, bk *BlockKey) {
	if c.State < CROP_MAX_STATE {
		c.State += 1
	}
	crop := NewBlockById(constants.Wheat.Value, c.State)
	w.SetBlock(bk.X, bk.Y, bk.Z, crop)
	blockChange := BlockChangeOutPacket{
		X:         bk.X,
		Y:         bk.Y,
		Z:         bk.Z,
		BlockType: crop.TypeId,
		BlockMeta: crop.Metadata,
	}
	w.BroadcastPacket(blockChange.Serialize())
}
