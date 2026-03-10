package packets

import (
	"log"

	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type WindowClickInPacket struct {
	WindowId     byte
	Slot         int16
	RightClick   byte
	ActionNumber int16
	Shift        bool
	ItemID       int16
	ItemCount    byte
	ItemUses     int16
}

func (p *WindowClickInPacket) Print() {
	slot := p.Slot
	rightClick := p.RightClick == 1
	shift := p.Shift
	log.Printf("Window click: slot=%d rightClick=%v shift=%v itemID=%d action=%d", slot, rightClick, shift, p.ItemID, p.ActionNumber)
}

func (p *WindowClickInPacket) GetItem() player.Item {
	return player.Item{
		TypeId: p.ItemID,
		Count:  p.ItemCount,
		Uses:   p.ItemUses,
	}
}

func ReadWindowClickInPacket(reader *packet.PacketReader) WindowClickInPacket {
	p := WindowClickInPacket{}
	_ = reader.GetPacketId()
	p.WindowId = reader.ReadByte()
	p.Slot = int16(reader.ReadShort())
	p.RightClick = reader.ReadByte()
	p.ActionNumber = int16(reader.ReadShort())
	p.Shift = reader.ReadBool()
	p.ItemID = int16(reader.ReadShort())
	// count and uses are only present when the player is holding an item
	if p.ItemID != -1 {
		p.ItemCount = reader.ReadByte()
		p.ItemUses = int16(reader.ReadShort())
	}
	return p
}
