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

type KeepAliveInPacket struct {
	packet.Packet
}

func ReadKeepAliveInPacket(reader *packet.PacketReader) KeepAliveInPacket {
	p := KeepAliveInPacket{}
	p.PacketId = reader.GetPacketId()
	return p
}

type KeepAliveOutPacket struct {
	packet.Packet
}

func (p *KeepAliveOutPacket) Serialize() []byte {
	return []byte{packet.KeepAlive}
}

type LoginRequestInPacket struct {
	packet.Packet
	ProtocolVersion int
	Username        string
	MapSeed         int64
	Dimension       byte
}

func ReadLoginRequestInPacket(reader *packet.PacketReader) LoginRequestInPacket {
	packet := LoginRequestInPacket{}
	packet.ProtocolVersion = reader.ReadInt()
	packet.Username = reader.ReadString16AndDecodeUTF16()
	packet.MapSeed = reader.ReadLong()
	packet.Dimension = reader.ReadByte()
	return packet
}

type LoginResponseOutPacket struct {
	EntityId  int
	MapSeed   int64
	Dimension byte
}

func (p *LoginResponseOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.LoginRequest)
	w.WriteInt32(int32(p.EntityId))
	w.WriteString16("") // write unknown attribute (possible server name?)
	w.WriteInt64(p.MapSeed)
	w.WriteByte(p.Dimension)
	return w.Bytes()
}
