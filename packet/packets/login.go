package packets

import "github.com/leNicDev/retromc/packet"

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
	w.WriteByte(packet.LoginRequest) // write packet id
	w.WriteInt32(int32(p.EntityId))  // write entity id
	w.WriteString16("")              // write unknown attribute (possible server name?)
	w.WriteInt64(p.MapSeed)          // write map seed
	w.WriteByte(p.Dimension)         // write dimension
	return w.Bytes()
}
