package packets

import "github.com/leNicDev/retromc/packet"

type DisconnectPacket struct {
	packet.Packet
	Reason string // string16
}

func ReadDisconnectPacket(reader *packet.PacketReader) DisconnectPacket {
	packet := DisconnectPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Reason = reader.ReadString16()
	return packet
}

type PreLoginPacket struct {
	packet.Packet
	Username       string // string16, C->S
	ConnectionHash string // string16, S->C
}

func ReadPreLoginPacket(reader *packet.PacketReader) PreLoginPacket {
	packet := PreLoginPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Username = reader.ReadString16()
	return packet
}

func (p *PreLoginPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.PreLogin)      // write packet id
	w.WriteString16(p.ConnectionHash) // write connection hash (no name authentication)
	return w.Bytes()
}

type KeepAlivePacket struct {
	packet.Packet
}

func ReadKeepAlivePacket(reader *packet.PacketReader) KeepAlivePacket {
	p := KeepAlivePacket{}
	p.PacketId = reader.GetPacketId()
	return p
}

func (p *KeepAlivePacket) Serialize() []byte {
	return []byte{packet.KeepAlive}
}

type LoginPacket struct {
	packet.Packet
	ProtocolVersion int
	Username        string
	MapSeed         int64
	Dimension       byte
	EntityId        int
}

func ReadLoginPacket(reader *packet.PacketReader) LoginPacket {
	packet := LoginPacket{}
	packet.ProtocolVersion = reader.ReadInt()
	packet.Username = reader.ReadString16AndDecodeUTF16()
	packet.MapSeed = reader.ReadLong()
	packet.Dimension = reader.ReadByte()
	return packet
}

func (p *LoginPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.Login)
	w.WriteInt32(int32(p.EntityId))
	w.WriteString16("") // write unknown attribute (possible server name?)
	w.WriteInt64(p.MapSeed)
	w.WriteByte(p.Dimension)
	return w.Bytes()
}
