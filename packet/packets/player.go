package packets

import (
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type SpawnPositionOutPacket struct {
	X int32
	Y int32
	Z int32
}

func (p *SpawnPositionOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SpawnPosition) 
	writer.WriteInt32(p.X)
	writer.WriteInt32(p.Y)
	writer.WriteInt32(p.Z)

	return writer.Bytes()
}

type PlayerAnimation struct {
	PlayerId int32
	Action   byte
}

type SetEquipmentOutPacket struct {
	EntityId      int32
	InventorySlot int16
	ItemId        int16
	ItemMetadata  int16
}

type SpawnPlayerEntityOutPacket struct {
	EntityId int32
	Username string
	X        int32
	Y        int32
	Z        int32
	Yaw      byte
	Pitch    byte
	HeldItem int16
}

func (p *SpawnPlayerEntityOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SpawnPlayerEntity)
	w.WriteInt32(p.EntityId)
	// log.Printf("Raw username bytes: % X\n", []byte(p.Username))
	// log.Printf("Raw username string: %q\n", p.Username)
	// log.Printf("Decoded username: %q\n", username)
	// log.Printf("UTF-16 bytes: % X\n", []byte(username))
	w.WriteString16(p.Username)
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteByte(p.Yaw)
	w.WriteByte(p.Pitch)
	w.WriteShort(uint16(p.HeldItem))
	return w.Bytes()
}

func (p *SetEquipmentOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SetEquipment)
	w.WriteInt32(p.EntityId)
	w.WriteShort(uint16(p.InventorySlot))
	w.WriteShort(uint16(p.ItemId))
	w.WriteShort(uint16(p.ItemMetadata))
	return w.Bytes()
}

func (p *PlayerAnimation) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.PlayerAnimation)
	w.WriteInt32(p.PlayerId)
	w.WriteByte(p.Action)
	return w.Bytes()
}

func ArmSwing(pl *player.Player) []byte {
	p := PlayerAnimation{
		PlayerId: int32(pl.EntityId),
		Action:   1,
	}
	return p.Serialize()
}

func SpawnPlayerEntityPacket(pl *player.Player) []byte {
	// The protocol encodes positions in entity space: 1 block = 32 units.
	p := SpawnPlayerEntityOutPacket{
		EntityId: int32(pl.EntityId),
		Username: pl.Username,
		X:        int32(pl.X * 32),
		Y:        int32(pl.Y * 32),
		Z:        int32(pl.Z * 32),
		Yaw:      byte(pl.Yaw),
		Pitch:    byte(pl.Pitch),
		HeldItem: 0,
	}
	return p.Serialize()
}

func SetEquipment(pl *player.Player, send func([]byte)) {
	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	data := map[int16]int16{
		0: heldItem.TypeId,
		1: -1,
		2: -1,
		3: -1,
		4: -1,
	}
	for slot, itemId := range data {
		p := &SetEquipmentOutPacket{
			EntityId:      int32(pl.EntityId),
			InventorySlot: slot,
			ItemId:        itemId,
			ItemMetadata:  0,
		}
		send(p.Serialize())
	}
}

type RespawnPacket struct {
	World byte
}

func ReadRespawnInPacket(reader *packet.PacketReader) RespawnPacket {
	packet := RespawnPacket{}
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

func ReadPlayerAnimationInPacket(reader *packet.PacketReader) PlayerAnimationInPacket {
	packet := PlayerAnimationInPacket{}
	packet.PacketId = reader.GetPacketId()
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

func ReadPlayerLookInPacket(reader *packet.PacketReader) PlayerLookInPacket {
	packet := PlayerLookInPacket{}
	packet.PacketId = reader.GetPacketId()
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

func ReadPlayerOnGroundInPacket(reader *packet.PacketReader) PlayerOnGroundInPacket {
	packet := PlayerOnGroundInPacket{}
	packet.PacketId = reader.GetPacketId()
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

func ReadPlayerPositionInPacket(reader *packet.PacketReader) PlayerPositionInPacket {
	packet := PlayerPositionInPacket{}
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

func ReadPlayerPositionAndLookInPacket(reader *packet.PacketReader) PlayerPositionAndLookInPacket {
	packet := PlayerPositionAndLookInPacket{}
	packet.PacketId = reader.GetPacketId()
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
	w.WriteByte(packet.PlayerPositionAndLook)
	w.WriteFloat64(p.X)                       
	w.WriteFloat64(p.Stance)                 
	w.WriteFloat64(p.Y)                      
	w.WriteFloat64(p.Z)                    
	w.WriteFloat32(p.Yaw)                    
	w.WriteFloat32(p.Pitch)                
	w.WriteBool(p.OnGround)                 
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

func ReadPlaceInPacket(reader *packet.PacketReader) PlayerBlockPlacementInPacket {
	packet := PlayerBlockPlacementInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte() // Direction
	packet.ItemId = int16(reader.ReadShort())
	if packet.ItemId >= 0 {
		packet.Amount = reader.ReadByte()
		packet.Damage = int16(reader.ReadShort())
	}
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

func ReadPlayerDiggingInPacket(reader *packet.PacketReader) PlayerDiggingInPacket {
	packet := PlayerDiggingInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Status = reader.ReadByte()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte()
	//log.Printf("Mine: %+v", packet)
	return packet
}

type HoldingChangeInPacket struct {
	packet.Packet
	Slot int16
}

func ReadHoldingChangeInPacket(reader *packet.PacketReader) HoldingChangeInPacket {
	packet := HoldingChangeInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Slot = int16(reader.ReadShort())
	//log.Printf("Holding change: %+v", packet)
	return packet
}

type BlockChangeOutPacket struct {
	X         int32
	Y         byte
	Z         int32
	BlockType byte
	BlockMeta byte
}

func (p *BlockChangeOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.BlockChange)
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteByte(p.BlockType)
	writer.WriteByte(p.BlockMeta)
	return writer.Bytes()
}
