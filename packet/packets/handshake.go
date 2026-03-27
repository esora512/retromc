package packets

import "github.com/leNicDev/retromc/packet"

type DisconnectInPacket struct {
	packet.Packet
	Reason string // string16
}

func ReadDisconnectInPacket(reader *packet.PacketReader) DisconnectInPacket {
	packet := DisconnectInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Reason = reader.ReadString16()
	return packet
}

type HandshakeInPacket struct {
	packet.Packet
	Username string // string16
}

func ReadHandshakeInPacket(reader *packet.PacketReader) HandshakeInPacket {
	packet := HandshakeInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Username = reader.ReadString16()
	return packet
}

type HandshakeOutPacket struct {
	ConnectionHash string // string16
}

func (p *HandshakeOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.Handshake)     // write packet id
	w.WriteString16(p.ConnectionHash) // write connection hash (no name authentication)
	return w.Bytes()
}
