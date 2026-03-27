package packets

import (
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type EntityPositionAndLookOutPacket struct {
	EntityId int32
	X        byte
	Y        byte
	Z        byte
	Yaw      byte
	Pitch    byte
}

type EntityPositionOutPacket struct {
	EntityId int32
	X        byte
	Y        byte
	Z        byte
}

type EntityLookOutPacket struct {
	EntityId int32
	Yaw      byte
	Pitch    byte
}

type EntityDespawnOutPacket struct {
	EntityId int32
}

func (p *EntityPositionAndLookOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityPositionAndRotation) // write packet id
	writer.WriteInt32(p.EntityId)                      // write entity id
	writer.WriteByte(p.X)                              // write x position
	writer.WriteByte(p.Y)                              // write y position
	writer.WriteByte(p.Z)                              // write z position
	writer.WriteByte(p.Yaw)                            // write yaw
	writer.WriteByte(p.Pitch)                          // write pitch
	return writer.Bytes()
}

func (p *EntityPositionOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityPosition) // write packet id
	writer.WriteInt32(p.EntityId)           // write entity id
	writer.WriteByte(p.X)                   // write x position
	writer.WriteByte(p.Y)                   // write y position
	writer.WriteByte(p.Z)                   // write z position
	return writer.Bytes()
}

func (p *EntityLookOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityLook) // write packet id
	writer.WriteInt32(p.EntityId)       // write entity id
	writer.WriteByte(p.Yaw)             // write yaw
	writer.WriteByte(p.Pitch)           // write pitch
	return writer.Bytes()
}

func (p *EntityDespawnOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.DespawnEntity) // write packet id
	writer.WriteInt32(p.EntityId)          // write entity id
	return writer.Bytes()
}

func PlayerEntityDespawnPacket(pl *player.Player) []byte {
	p := EntityDespawnOutPacket{
		EntityId: int32(pl.EntityId),
	}
	return p.Serialize()
}

func PlayerEntityPositionAndLookPacket(pl *player.Player, x, y, z, yaw, pitch float64, world *level.World) []byte {
	dX := int32((x - pl.X) * 32)
	dY := int32((y - pl.Y) * 32)
	dZ := int32((z - pl.Z) * 32)
	dYaw := int32(yaw * 256 / 360)
	dPitch := int32(pitch * 256 / 360)

	p := EntityPositionAndLookOutPacket{
		EntityId: int32(pl.EntityId),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func PlayerEntityPositionPacket(pl *player.Player, x, y, z float64, world *level.World) []byte {
	dX := int32((x - pl.X) * 32)
	dY := int32((y - pl.Y) * 32)
	dZ := int32((z - pl.Z) * 32)

	p := EntityPositionOutPacket{
		EntityId: int32(pl.EntityId),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	return p.Serialize()
}

func PlayerEntityLookPacket(pl *player.Player, yaw, pitch float64, world *level.World) []byte {
	dYaw := int32(yaw * 256 / 360)
	dPitch := int32(pitch * 256 / 360)

	p := EntityLookOutPacket{
		EntityId: int32(pl.EntityId),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
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
