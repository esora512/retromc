package packets

import (
	"math"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type UpdateSignPacket struct {
	packet.Packet
	X     int32
	Y     int16
	Z     int32
	Text1 string
	Text2 string
	Text3 string
	Text4 string
}

func ReadUpdateSignPacket(reader *packet.PacketReader) UpdateSignPacket {
	packet := UpdateSignPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.X = reader.ReadInt32()
	packet.Y = int16(reader.ReadShort())
	packet.Z = reader.ReadInt32()
	packet.Text1 = reader.ReadString16AndDecodeUTF16()
	packet.Text2 = reader.ReadString16AndDecodeUTF16()
	packet.Text3 = reader.ReadString16AndDecodeUTF16()
	packet.Text4 = reader.ReadString16AndDecodeUTF16()
	return packet
}

func (p *UpdateSignPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.UpdateSign)
	w.WriteInt32(p.X)
	w.WriteShort(uint16(p.Y))
	w.WriteInt32(p.Z)
	w.WriteString16(p.Text1)
	w.WriteString16(p.Text2)
	w.WriteString16(p.Text3)
	w.WriteString16(p.Text4)
	return w.Bytes()
}

type SpawnPositionPacket struct {
	X int32
	Y int32
	Z int32
}

func (p *SpawnPositionPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SetSpawnPosition)
	writer.WriteInt32(p.X)
	writer.WriteInt32(p.Y)
	writer.WriteInt32(p.Z)

	return writer.Bytes()
}

type AnimationPacket struct {
	packet.Packet
	PlayerId  int32
	Animation byte
}

type SetEquipmentPacket struct {
	EntityId      int32
	InventorySlot int16
	ItemId        int16
	ItemMetadata  int16
}

type SpawnPlayerPacket struct {
	EntityId int32
	Username string
	X        int32
	Y        int32
	Z        int32
	Yaw      byte
	Pitch    byte
	HeldItem int16
}

func (p *SpawnPlayerPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SpawnPlayer)
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

func (p *SetEquipmentPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SetEquipment)
	w.WriteInt32(p.EntityId)
	w.WriteShort(uint16(p.InventorySlot))
	w.WriteShort(uint16(p.ItemId))
	w.WriteShort(uint16(p.ItemMetadata))
	return w.Bytes()
}

func (p *AnimationPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.Animation)
	w.WriteInt32(p.PlayerId)
	w.WriteByte(p.Animation)
	return w.Bytes()
}

func ArmSwing(pl *player.Player) []byte {
	p := AnimationPacket{
		PlayerId:  int32(pl.EntityId),
		Animation: 1,
	}
	return p.Serialize()
}

func quantizeSpawnVelocity(v float64) int8 {
	return int8(v * 128.0)
}

func NewSpawnItem(d *level.DroppedItem) []byte {
	p := SpawnItemPacket{
		EntityId: d.EntityId,
		ItemId:   int16(d.ItemId),
		Amount:   d.Amount,
		Metadata: d.Metadata,
		X:        int32(math.Floor(d.MovementState.X * 32)),
		Y:        int32(math.Floor(d.MovementState.Y * 32)),
		Z:        int32(math.Floor(d.MovementState.Z * 32)),
		Pitch:    byte(quantizeSpawnVelocity(d.MovementState.VelocityX)),
		Yaw:      byte(quantizeSpawnVelocity(d.MovementState.VelocityY)),
		Roll:     byte(quantizeSpawnVelocity(d.MovementState.VelocityZ)),
	}
	return p.Serialize()
}

func NewSpawnPlayerPacket(pl *player.Player) []byte {
	// The protocol encodes positions in entity space: 1 block = 32 units.
	//log.Printf("Spawn %s at x=%f, y=%f, z=%f", pl.Username, pl.X, pl.Y, pl.Z)
	p := SpawnPlayerPacket{
		EntityId: int32(pl.EntityId),
		Username: pl.Username,
		X:        int32(pl.X * 32),
		Y:        int32(pl.Y * 32),
		Z:        int32(pl.Z * 32),
		Yaw:      byte(math.Round(float64(pl.Yaw) / 360.0 * 255.0)),
		Pitch:    byte(math.Round(float64(pl.Pitch) / 360 * 255)),
		HeldItem: 0,
	}
	return p.Serialize()
}

func NewSpawnObjectPacket(e constants.Entity) []byte {
	// NOTE: Bad practice but we wing it...
	rideable, _ := e.(*entities.RideableEntity)
	p := SpawnObjectPacket{
		EntityId:      e.GetEntityId(),
		ObjectType:    rideable.ObjectType,
		X:             int32(rideable.X * 32),
		Y:             int32(rideable.Y * 32),
		Z:             int32(rideable.Z * 32),
		VelocityX:     int16(rideable.VelocityX),
		VelocityY:     int16(rideable.VelocityY),
		VelocityZ:     int16(rideable.VelocityZ),
		OwnerEntityId: rideable.OwnerEntityId,
	}
	return p.Serialize()
}

func SetEquipment(pl *player.Player, send func([]byte)) {
	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	data := map[int16]int16{
		0: heldItem.TypeId,
		1: pl.Inventory.Items[8].TypeId,
		2: pl.Inventory.Items[7].TypeId,
		3: pl.Inventory.Items[6].TypeId,
		4: pl.Inventory.Items[5].TypeId,
	}
	for slot, itemId := range data {
		p := &SetEquipmentPacket{
			EntityId:      int32(pl.EntityId),
			InventorySlot: slot,
			ItemId:        itemId,
			ItemMetadata:  0,
		}
		send(p.Serialize())
	}
}

func SetEquipment2(pl *player.Player, send func([]byte) (int, error)) {
	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	data := map[int16]int16{
		0: heldItem.TypeId,
		1: pl.Inventory.Items[8].TypeId,
		2: pl.Inventory.Items[7].TypeId,
		3: pl.Inventory.Items[6].TypeId,
		4: pl.Inventory.Items[5].TypeId,
	}
	for slot, itemId := range data {
		p := &SetEquipmentPacket{
			EntityId:      int32(pl.EntityId),
			InventorySlot: slot,
			ItemId:        itemId,
			ItemMetadata:  0,
		}
		send(p.Serialize())
	}
}

func SetEquipment3(pl *player.Player) {
	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	data := map[int16]int16{
		0: heldItem.TypeId,
		1: pl.Inventory.Items[8].TypeId,
		2: pl.Inventory.Items[7].TypeId,
		3: pl.Inventory.Items[6].TypeId,
		4: pl.Inventory.Items[5].TypeId,
	}
	for slot, itemId := range data {
		p := &SetEquipmentPacket{
			EntityId:      int32(pl.EntityId),
			InventorySlot: slot,
			ItemId:        itemId,
			ItemMetadata:  0,
		}
		pl.Connection.Write(p.Serialize())
	}
}

type RespawnPacket struct {
	World byte
}

func ReadRespawnPacket(reader *packet.PacketReader) RespawnPacket {
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

type SetHealthPacket struct {
	Health uint16
}

func (p *SetHealthPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SetHealth)
	w.WriteShort(p.Health)
	return w.Bytes()
}

func ReadAnimationPacket(reader *packet.PacketReader) AnimationPacket {
	packet := AnimationPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.PlayerId = reader.ReadInt32()
	packet.Animation = reader.ReadByte()
	return packet
}

type PlayerRotationPacket struct {
	packet.Packet
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func ReadPlayerRotationPacket(reader *packet.PacketReader) PlayerRotationPacket {
	packet := PlayerRotationPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Yaw = reader.ReadFloat32()
	packet.Pitch = reader.ReadFloat32()
	packet.OnGround = reader.ReadBool()
	//log.Printf("Player look: %+v", packet)
	return packet
}

type PlayerMovementPacket struct {
	packet.Packet
	OnGround bool
}

func ReadPlayerMovementPacket(reader *packet.PacketReader) PlayerMovementPacket {
	packet := PlayerMovementPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.OnGround = reader.ReadBool()
	return packet
}

type PlayerPositionPacket struct {
	packet.Packet
	X        float64
	Y        float64
	Stance   float64
	Z        float64
	OnGround bool
}

func (p *PlayerPositionPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.PlayerPosition)
	w.WriteFloat64(p.X)
	w.WriteFloat64(p.Y)
	w.WriteFloat64(p.Stance)
	w.WriteFloat64(p.Z)
	w.WriteBool(p.OnGround)
	return w.Bytes()
}

func ReadPlayerPositionPacket(reader *packet.PacketReader) PlayerPositionPacket {
	packet := PlayerPositionPacket{}
	packet.X = reader.ReadFloat64()
	packet.Y = reader.ReadFloat64()
	packet.Stance = reader.ReadFloat64()
	packet.Z = reader.ReadFloat64()
	packet.OnGround = reader.ReadBool()
	return packet
}

type PlayerPositionAndRotationPacket struct {
	packet.Packet
	X        float64
	Y        float64
	Stance   float64
	Z        float64
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func ReadPlayerPositionAndRotationPacket(reader *packet.PacketReader) PlayerPositionAndRotationPacket {
	packet := PlayerPositionAndRotationPacket{}
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

func (p *PlayerPositionAndRotationPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.PlayerPositionAndRotation)
	w.WriteFloat64(p.X)
	w.WriteFloat64(p.Stance)
	w.WriteFloat64(p.Y)
	w.WriteFloat64(p.Z)
	w.WriteFloat32(p.Yaw)
	w.WriteFloat32(p.Pitch)
	w.WriteBool(p.OnGround)
	return w.Bytes()
}

type PlaceBlockPacket struct {
	packet.Packet
	X      int32
	Y      byte
	Z      int32
	Face   byte
	ItemId int16
	Amount byte
	Damage int16 //short
}

func ReadPlaceBlockPacket(reader *packet.PacketReader) PlaceBlockPacket {
	packet := PlaceBlockPacket{}
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

type MineBlockPacket struct {
	packet.Packet
	Status byte
	X      int32
	Y      byte
	Z      int32
	Face   byte
}

func ReadPlayerMineBlockPacket(reader *packet.PacketReader) MineBlockPacket {
	packet := MineBlockPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Status = reader.ReadByte()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte()
	return packet
}

type SetHotbarSlotPacket struct {
	packet.Packet
	Slot int16
}

func ReadSetHotbarSlot(reader *packet.PacketReader) SetHotbarSlotPacket {
	packet := SetHotbarSlotPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Slot = int16(reader.ReadShort())
	return packet
}

type SetBlockPacket struct {
	X         int32
	Y         byte
	Z         int32
	BlockType byte
	BlockMeta byte
}

func (p *SetBlockPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SetBlock)
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteByte(p.BlockType)
	writer.WriteByte(p.BlockMeta)
	return writer.Bytes()
}

func BroadcastBlockChange(w *level.World, x, y, z int32, blockType, blockMeta byte) {
	p := SetBlockPacket{
		X:         x,
		Y:         byte(y),
		Z:         z,
		BlockType: blockType,
		BlockMeta: blockMeta,
	}
	w.BroadcastPacket(p.Serialize())
}

type PlayerInputPacket struct {
	packet.Packet
	StrafeDirection  float64
	ForwardDirection float64
	Pitch            float64
	Yaw              float64
	Jumping          bool
	Sneaking         bool
}

func ReadPlayerInputPacket(reader *packet.PacketReader) PlayerInputPacket {
	packet := PlayerInputPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.StrafeDirection = reader.ReadFloat64()
	packet.ForwardDirection = reader.ReadFloat64()
	packet.Pitch = reader.ReadFloat64()
	packet.Yaw = reader.ReadFloat64()
	packet.Jumping = reader.ReadBool()
	packet.Sneaking = reader.ReadBool()
	return packet
}

type SetMultipleBlocksPacket struct {
	ChunkX      int32
	ChunkZ      int32
	NumOfBlocks uint16
	BlockCoords []uint16
	BlockTypes  []byte
	Metadata    []byte
}

func (p *SetMultipleBlocksPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SetMultipleBlocks)
	writer.WriteInt32(p.ChunkX)
	writer.WriteInt32(p.ChunkZ)
	writer.WriteShort(p.NumOfBlocks)
	writer.WriteShortArray(p.BlockCoords)
	writer.Write(p.BlockTypes)
	writer.Write(p.Metadata)
	return writer.Bytes()
}

func BroadcastMultiBlockChange(world *level.World, chunkX, chunkZ int32, numOfBlocks uint16, blockCoords []uint16, blockTypes, metadata []byte) {
	p := SetMultipleBlocksPacket{
		ChunkX:      chunkX,
		ChunkZ:      chunkZ,
		NumOfBlocks: numOfBlocks,
		BlockCoords: blockCoords,
		BlockTypes:  blockTypes,
		Metadata:    metadata,
	}
	world.BroadcastPacket(p.Serialize())
}

type InteractWithBlockPacket struct {
	EntityId int32
	Type     byte
	X        int32
	Y        byte
	Z        int32
}

func (p *InteractWithBlockPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.InteractWithBlock)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.Type)
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	return writer.Bytes()
}

func NewInteractWithBlockPacket(entityId int32, bedType byte, x int32, y byte, z int32) []byte {
	p := InteractWithBlockPacket{
		EntityId: entityId,
		Type:     bedType,
		X:        x,
		Y:        y,
		Z:        z,
	}
	return p.Serialize()
}
