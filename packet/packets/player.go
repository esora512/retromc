package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type RespawnPacket struct {
	World byte
}

func ReadRespawnInPacket(data *[]byte) RespawnPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := RespawnPacket{}
	_ = reader.ReadPacketId()
	packet.World = reader.ReadByte()

	return packet
}

func (p *RespawnPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.Respawn)
	w.WriteByte(p.World)
	return w.Bytes()
}

type SetHealthOutPacket struct {
	Health float32
}

func (p *SetHealthOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SetHealth)
	w.WriteFloat32(p.Health)
	return w.Bytes()
}

type PlayerAnimationInPacket struct {
	packet.Packet
	PlayerId  int
	Animation byte
}

func ReadPlayerAnimationInPacket(data *[]byte) PlayerAnimationInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerAnimationInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.PlayerId = reader.ReadInt()
	packet.Animation = reader.ReadByte()
	//log.Printf("Player animation: %+v", packet)
	return packet
}

type PlayerLookInPacket struct {
	packet.Packet
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func ReadPlayerLookInPacket(data *[]byte) PlayerLookInPacket {
	reader := packet.PacketReader{
		Data: data,
	}
	packet := PlayerLookInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.Yaw = reader.ReadFloat32()
	packet.Pitch = reader.ReadFloat32()
	packet.OnGround = reader.ReadBool()
	//log.Printf("Player look: %+v", packet)

	return packet
}

type PlayerOnGroundInPacket struct {
	packet.Packet
	OnGround bool
}

func ReadPlayerOnGroundInPacket(data *[]byte) PlayerOnGroundInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerOnGroundInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.OnGround = reader.ReadBool()
	return packet
}

type PlayerPositionInPacket struct {
	packet.Packet
	X        float64
	Y        float64
	Stance   float64
	Z        float64
	OnGround bool
}

type PlayerPositionOutPacket struct {
	packet.Packet
	X        float64
	Y        float64
	Stance   float64
	Z        float64
	OnGround bool
}

func (p *PlayerPositionOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.PlayerPosition) // write packet id
	w.WriteFloat64(p.X)                // write x position
	w.WriteFloat64(p.Y)                // write y position
	w.WriteFloat64(p.Stance)           // write stance
	w.WriteFloat64(p.Z)                // write z position
	w.WriteBool(p.OnGround)            // write on ground
	return w.Bytes()
}

func ReadPlayerPositionInPacket(data *[]byte) PlayerPositionInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerPositionInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.X = reader.ReadFloat64()
	packet.Y = reader.ReadFloat64()
	packet.Stance = reader.ReadFloat64()
	packet.Z = reader.ReadFloat64()
	packet.OnGround = reader.ReadBool()

	return packet
}

type PlayerPositionAndLookInPacket struct {
	packet.Packet
	X        float64
	Y        float64
	Stance   float64
	Z        float64
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func ReadPlayerPositionAndLookInPacket(data *[]byte) PlayerPositionAndLookInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerPositionAndLookInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.X = reader.ReadFloat64()
	packet.Y = reader.ReadFloat64()
	packet.Stance = reader.ReadFloat64()
	packet.Z = reader.ReadFloat64()
	packet.Yaw = reader.ReadFloat32()
	packet.Pitch = reader.ReadFloat32()
	packet.OnGround = reader.ReadBool()

	return packet
}

type PlayerPositionAndLookOutPacket struct {
	X        float64
	Y        float64
	Stance   float64
	Z        float64
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func (p *PlayerPositionAndLookOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.PlayerPositionAndLook) // write packet id
	w.WriteFloat64(p.X)                       // write x position
	w.WriteFloat64(p.Stance)                  // write stance
	w.WriteFloat64(p.Y)                       // write y position
	w.WriteFloat64(p.Z)                       // write z position
	w.WriteFloat32(p.Yaw)                     // write yaw
	w.WriteFloat32(p.Pitch)                   // write pitch
	w.WriteBool(p.OnGround)                   // write on ground
	return w.Bytes()
}

type PlayerBlockPlacementInPacket struct {
	packet.Packet
	X      int32
	Y      byte
	Z      int32
	Face   byte
	ItemId int16
	Amount byte
	Damage int16 //short
}

func ReadPlaceInPacket(data *[]byte) PlayerBlockPlacementInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerBlockPlacementInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte() // Direction
	packet.ItemId = int16(reader.ReadShort())
	packet.Amount = reader.ReadByte()
	packet.Damage = int16(reader.ReadShort())
	return packet
}

type PlayerDiggingInPacket struct {
	packet.Packet
	Status byte
	X      int32
	Y      byte
	Z      int32
	Face   byte
}

func ReadPlayerDiggingInPacket(data *[]byte) PlayerDiggingInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerDiggingInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.Status = reader.ReadByte()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte()
	//log.Printf("Mine: %+v", packet)

	return packet
}
