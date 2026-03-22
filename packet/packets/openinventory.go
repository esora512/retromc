package packets

import "github.com/leNicDev/retromc/packet"

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

func (p *OpenInventoryOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.OpenInventory)
	w.WriteByte(p.WindowID)
	w.WriteByte(p.Type)
	w.WriteString8(p.Title)
	w.WriteByte(p.Size)
	return w.Bytes()
}
