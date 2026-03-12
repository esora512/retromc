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
		WindowID: 0x01,
		Type:     0x01,
		Title:    "CraftingTable",
		Size:     0x09,
	}
	return p
}

func (p *OpenInventoryOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(p.WindowID)
	w.WriteByte(p.Type)
	w.WriteString8(p.Title)
	w.WriteByte(p.Size)
	return w.Bytes()
}
