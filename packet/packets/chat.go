package packets

import "github.com/leNicDev/retromc/packet"

type ChatMessagePacket struct {
	packet.Packet
	Message string
}

func ReadChatMessageInPacket(reader *packet.PacketReader) ChatMessagePacket {
	packet := ChatMessagePacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Message = reader.ReadString16AndDecodeUTF16()
	return packet
}

func (p *ChatMessagePacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.ChatMessage)
	w.WriteString16(p.Message)
	return w.Bytes()
}
