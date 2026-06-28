package packets

import (
	"log"

	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type SetSlotOutPacket struct {
	WindowId byte
	Slot     int16
	Item     inventory.Item
}

func (p *SetSlotOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SetSlot)
	writer.WriteByte(p.WindowId)
	writer.WriteShort(uint16(p.Slot))
	writer.Write(p.Item.Serialize())
	return writer.Bytes()
}

type WindowItemsOutPacket struct {
	WindowId byte
	Count    int16
	Payload  inventory.Inventory
}

func (p *WindowItemsOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.WindowItems)
	writer.WriteByte(p.WindowId)
	writer.WriteInt16(p.Count)
	writer.Write(p.Payload.Serialize())
	return writer.Bytes()
}

type WindowClickInPacket struct {
	WindowId     byte
	Slot         int16
	RightClick   byte
	ActionNumber int16
	Shift        bool
	ItemID       int16
	ItemCount    byte
	ItemUses     uint16
}

func (p *WindowClickInPacket) Print() {
	slot := p.Slot
	rightClick := p.RightClick == 1
	shift := p.Shift
	log.Printf("Window click: slot=%d rightClick=%v shift=%v itemID=%d action=%d", slot, rightClick, shift, p.ItemID, p.ActionNumber)
}

func (p *WindowClickInPacket) GetItem() inventory.Item {
	return inventory.Item{
		TypeId:   p.ItemID,
		Count:    p.ItemCount,
		Metadata: p.ItemUses,
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
		p.ItemUses = uint16(reader.ReadShort())
	}
	return p
}

type CloseWindowInPacket struct {
	packet.Packet
	WindowId byte
}

func ReadCloseWindowInPacket(reader *packet.PacketReader, pl *player.Player) CloseWindowInPacket {
	p := CloseWindowInPacket{}
	p.PacketId = reader.GetPacketId()
	p.WindowId = reader.ReadByte()
	if p.WindowId == 1 {
		pl.Workbench.ClearGrid()
	}
	return p
}

type OpenInventoryOutPacket struct {
	WindowID byte
	Type     byte
	Title    string
	Size     byte
}

func NewCraftingTable() OpenInventoryOutPacket {
	p := OpenInventoryOutPacket{
		WindowID: byte(1),
		Type:     1,
		Title:    "Crafting",
		Size:     9,
	}
	return p
}

func NewChest(size byte) OpenInventoryOutPacket {
	p := OpenInventoryOutPacket{
		WindowID: byte(1),
		Type:     0,
		Title:    "Chest",
		Size:     size,
	}
	return p
}

func NewDispenser() OpenInventoryOutPacket {
	p := OpenInventoryOutPacket{
		WindowID: byte(1),
		Type:     3,
		Title:    "Dispenser",
		Size:     9,
	}
	return p
}

func NewFurnace() OpenInventoryOutPacket {
	p := OpenInventoryOutPacket{
		WindowID: byte(1),
		Type:     2,
		Title:    "Furnace",
		Size:     3,
	}
	return p
}

func (p *OpenInventoryOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.OpenInventory)
	w.WriteByte(p.WindowID)
	w.WriteByte(p.Type)
	w.WriteString8(p.Title)
	w.WriteByte(p.Size)
	return w.Bytes()
}

type ContainerDataOutPacket struct {
	WindowID byte
	Type     int16
	Value    int16
}

func (p *ContainerDataOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.ContainerData)
	w.WriteByte(p.WindowID)
	w.WriteShort(uint16(p.Type))
	w.WriteShort(uint16(p.Value))
	return w.Bytes()
}