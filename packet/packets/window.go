package packets

import (
	"log"

	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type SetSlotPacket struct {
	WindowId byte
	Slot     int16
	Item     inventory.Item
}

func (p *SetSlotPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SetSlot)
	writer.WriteByte(p.WindowId)
	writer.WriteShort(uint16(p.Slot))
	writer.Write(p.Item.Serialize())
	return writer.Bytes()
}

type FillContainerPacket struct {
	WindowId byte
	Count    int16
	Payload  inventory.Inventory
}

func (p *FillContainerPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.FillContainer)
	writer.WriteByte(p.WindowId)
	writer.WriteInt16(p.Count)
	writer.Write(p.Payload.Serialize())
	return writer.Bytes()
}

type ClickSlotPacket struct {
	WindowId     byte
	Slot         int16
	RightClick   byte
	ActionNumber int16
	Shift        bool
	ItemID       int16
	ItemCount    byte
	ItemUses     uint16
}

func (p *ClickSlotPacket) Print() {
	slot := p.Slot
	rightClick := p.RightClick == 1
	shift := p.Shift
	log.Printf("Window click: slot=%d rightClick=%v shift=%v itemID=%d action=%d", slot, rightClick, shift, p.ItemID, p.ActionNumber)
}

func (p *ClickSlotPacket) GetItem() inventory.Item {
	return inventory.Item{
		TypeId:   p.ItemID,
		Count:    p.ItemCount,
		Metadata: p.ItemUses,
	}
}

func ReadClickSlotPacket(reader *packet.PacketReader) ClickSlotPacket {
	p := ClickSlotPacket{}
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
		p.ItemUses = uint16(reader.ReadShort())
	}
	return p
}

type CloseContainerPacket struct {
	packet.Packet
	WindowId byte
}

func ReadCloseContainerPacket(reader *packet.PacketReader, pl *player.Player) CloseContainerPacket {
	p := CloseContainerPacket{}
	p.PacketId = reader.GetPacketId()
	p.WindowId = reader.ReadByte()
	if p.WindowId == 1 {
		pl.Workbench.ClearGrid()
	}
	return p
}

type OpenContainerPacket struct {
	WindowID byte
	Type     byte
	Title    string
	Size     byte
}

func NewCraftingTable() OpenContainerPacket {
	p := OpenContainerPacket{
		WindowID: byte(1),
		Type:     1,
		Title:    "Crafting",
		Size:     9,
	}
	return p
}

func NewChest(size byte) OpenContainerPacket {
	p := OpenContainerPacket{
		WindowID: byte(1),
		Type:     0,
		Title:    "Chest",
		Size:     size,
	}
	return p
}

func NewDispenser() OpenContainerPacket {
	p := OpenContainerPacket{
		WindowID: byte(1),
		Type:     3,
		Title:    "Dispenser",
		Size:     9,
	}
	return p
}

func NewFurnace() OpenContainerPacket {
	p := OpenContainerPacket{
		WindowID: byte(1),
		Type:     2,
		Title:    "Furnace",
		Size:     3,
	}
	return p
}

func (p *OpenContainerPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.OpenContainer)
	w.WriteByte(p.WindowID)
	w.WriteByte(p.Type)
	w.WriteString8(p.Title)
	w.WriteByte(p.Size)
	return w.Bytes()
}

type ContainerDataPacket struct {
	WindowID byte
	Type     int16
	Value    int16
}

func (p *ContainerDataPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.ContainerData)
	w.WriteByte(p.WindowID)
	w.WriteShort(uint16(p.Type))
	w.WriteShort(uint16(p.Value))
	return w.Bytes()
}
