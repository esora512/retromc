package packets

import "github.com/leNicDev/retromc/packet"

type EntityRelativePositionAndLookOutPacket struct {
	EntityId int16
	X        byte
	Y        byte
	Z        byte
	Yaw      byte
	Pitch    byte
}

func (p *EntityRelativePositionAndLookOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityRelativePositionAndLook) // write packet id
	writer.WriteInt16(p.EntityId)                          // write entity id
	writer.WriteByte(p.X)                                  // write x position
	writer.WriteByte(p.Y)                                  // write y position
	writer.WriteByte(p.Z)                                  // write z position
	writer.WriteByte(p.Yaw)                                // write yaw
	writer.WriteByte(p.Pitch)                              // write pitch
	return writer.Bytes()
}

type EntityActionInPacket struct {
	packet.Packet
	EntityId int
	ActionId byte
}

func ReadEntityActionInPacket(reader *packet.PacketReader) EntityActionInPacket {
	packet := EntityActionInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.EntityId = reader.ReadInt()
	packet.ActionId = reader.ReadByte()
	//log.Printf("Entity action: %+v", packet)

	return packet
}

type InteractWithEntityOutPacket struct {
	packet.Packet
	EntityId int32
	PlayerId int32
	Attack   bool // true = left click, false = right click
}

func ReadInteractWithEntityInPacket(reader *packet.PacketReader) InteractWithEntityOutPacket {
	packet := InteractWithEntityOutPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.PlayerId = reader.ReadInt32()
	packet.EntityId = reader.ReadInt32()
	packet.Attack = reader.ReadBool()
	return packet
}
