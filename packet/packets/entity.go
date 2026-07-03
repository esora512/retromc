package packets

import (
	"math"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type EntityVelocity struct {
	EntityId int32
	Vx       int16
	Vy       int16
	Vz       int16
}

type TeleportEntity struct {
	EntityId int32
	X        int32
	Y        int32
	Z        int32
	Yaw      byte
	Pitch    byte
}

type AddPassenger struct {
	Passenger int32
	Vehicle   int32
}

type SpawnObject struct {
	EntityId      int32
	ObjectType    byte
	X             int32
	Y             int32
	Z             int32
	OwnerEntityId int32
	VelocityX     int16
	VelocityY     int16
	VelocityZ     int16
}

type SpawnItem struct {
	EntityId int32
	ItemId   int16
	Amount   byte
	Metadata byte
	X        int32
	Y        int32
	Z        int32
	Yaw      byte
	Pitch    byte
	Roll     byte
}

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

type EntityEventOutPacket struct {
	EntityId int32
	Action   byte
}

type CollectItemOutPacket struct {
	ItemId      int32
	CollectorId int32
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (p *CollectItemOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.CollectItem)
	w.WriteInt32(p.ItemId)
	w.WriteInt32(p.CollectorId)
	return w.Bytes()
}

func (p *SpawnItem) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SpawnItem)
	w.WriteInt32(p.EntityId)
	w.WriteShort(uint16(p.ItemId))
	w.WriteByte(p.Amount)
	w.WriteShort(uint16(p.Metadata))
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteByte(p.Yaw)
	w.WriteByte(p.Pitch)
	w.WriteByte(p.Roll)
	return w.Bytes()
}

func (p *EntityVelocity) Serialize() []byte {
	vx := clamp(float64(p.Vx), -3.9, 3.9) * 8000
	vy := clamp(float64(p.Vy), -3.9, 3.9) * 8000
	vz := clamp(float64(p.Vz), -3.9, 3.9) * 8000

	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityVelocity)
	writer.WriteInt32(p.EntityId)
	writer.WriteShort(uint16(vx))
	writer.WriteShort(uint16(vy))
	writer.WriteShort(uint16(vz))
	return writer.Bytes()
}

func (p *EntityPositionAndLookOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityPositionAndRotation)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.X)
	writer.WriteByte(p.Y)
	writer.WriteByte(p.Z)
	writer.WriteByte(p.Yaw)
	writer.WriteByte(p.Pitch)
	return writer.Bytes()
}

func (p *EntityPositionOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityPosition)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.X)
	writer.WriteByte(p.Y)
	writer.WriteByte(p.Z)
	return writer.Bytes()
}

func (p *EntityLookOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityLook)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.Yaw)
	writer.WriteByte(p.Pitch)
	return writer.Bytes()
}

func (p *EntityDespawnOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.DespawnEntity)
	writer.WriteInt32(p.EntityId)
	return writer.Bytes()
}

func (p *SpawnObject) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SpawnObject)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.ObjectType)
	writer.WriteInt32(p.X)
	writer.WriteInt32(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteInt32(p.OwnerEntityId)
	writer.WriteInt16(p.VelocityX)
	writer.WriteInt16(p.VelocityY)
	writer.WriteInt16(p.VelocityZ)
	return writer.Bytes()
}

func (p *AddPassenger) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.AddPassenger)
	writer.WriteInt32(p.Passenger)
	writer.WriteInt32(p.Vehicle)
	return writer.Bytes()
}

func (p *TeleportEntity) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.TeleportEntity)
	writer.WriteInt32(p.EntityId)
	writer.WriteInt32(p.X)
	writer.WriteInt32(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteByte(p.Yaw)
	writer.WriteByte(p.Pitch)
	return writer.Bytes()
}

func AlicesRidesBob(alice, bob int32) []byte {
	p := AddPassenger{
		Passenger: alice,
		Vehicle:   bob,
	}
	return p.Serialize()
}

func PlayerEntityDespawnPacket(pl *player.Player) []byte {
	p := EntityDespawnOutPacket{
		EntityId: int32(pl.EntityId),
	}
	return p.Serialize()
}

func TeleportPlayerPacket(pl *player.Player, x, y, z, yaw, pitch float64, world *level.World) []byte {
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))
	p := TeleportEntity{
		EntityId: pl.GetEntityId(),
		X:        encX,
		Y:        encY,
		Z:        encZ,
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func PlayerEntityPositionAndLookPacket(pl *player.Player, x, y, z, yaw, pitch float64, world *level.World) []byte {
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))
	dX := encX - int32(math.Floor(pl.X*32))
	dY := encY - int32(math.Floor(pl.Y*32))
	dZ := encZ - int32(math.Floor(pl.Z*32))
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))

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
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))
	dX := encX - int32(math.Floor(pl.X*32))
	dY := encY - int32(math.Floor(pl.Y*32))
	dZ := encZ - int32(math.Floor(pl.Z*32))

	p := EntityPositionOutPacket{
		EntityId: int32(pl.EntityId),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	return p.Serialize()
}

func PlayerEntityLookPacket(pl *player.Player, yaw, pitch float64, world *level.World) []byte {
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))

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

type EntityMetadata struct {
	EntityId int32
	Metadata []byte
}

func sneakMetadata(sneaking bool) []byte {
	var flags byte = 0x00
	if sneaking {
		flags = 0x02
	}
	metadataType := byte(0)                       // 0 = byte type
	metadataIndex := byte(0)                      // 0 = entity flags field
	header := (metadataType << 5) | metadataIndex // encode type and index into single byte

	// S->C: Contains byte of id flag with value 0x02 if sneaking, 0x00 if not sneaking
	return []byte{
		header,
		flags, // 0x02 = sneaking, 0x00 = not sneaking
		0x7F,  // end of metadata
	}
}

func ridingMetadata(riding bool) []byte {
	var flags byte = 0x00
	if riding {
		flags = 0x04
	}
	metadataType := byte(0)                       // 0 = byte type
	metadataIndex := byte(0)                      // 0 = entity flags field
	header := (metadataType << 5) | metadataIndex // encode type and index into single byte
	return []byte{
		header,
		flags,
		0x7F,
	}
}

func (p *EntityMetadata) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.EntityMetadata)
	w.WriteInt32(p.EntityId)
	w.Write(p.Metadata)
	return w.Bytes()
}

func PlayerEntityMetadataPacketSneak(pl *player.Player, sneaking bool) []byte {
	p := EntityMetadata{
		EntityId: int32(pl.EntityId),
		Metadata: sneakMetadata(sneaking),
	}
	return p.Serialize()
}

func PlayerEntityMetadataPacketRiding(pl *player.Player, riding bool) []byte {
	p := EntityMetadata{
		EntityId: int32(pl.EntityId),
		Metadata: ridingMetadata(riding),
	}
	return p.Serialize()
}

func (p *EntityEventOutPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.EntityEvent)
	w.WriteInt32(p.EntityId)
	w.WriteByte(p.Action)
	return w.Bytes()
}

func CollectItem(itemId, collectorId int32) []byte {
	p := CollectItemOutPacket{ItemId: itemId, CollectorId: collectorId}
	return p.Serialize()
}

func SpawnDroppedItem(w *level.World, itemId int16, amount, meta byte, x, y, z int32, yaw, pitch, roll byte) []byte {
	entityId := w.AddDroppedItem(x, y, z, int32(itemId), amount, meta)
	p := SpawnItem{
		EntityId: entityId,
		ItemId:   itemId,
		Amount:   amount,
		Metadata: meta,
		X:        x,
		Y:        y,
		Z:        z,
		Yaw:      yaw,
		Pitch:    pitch,
		Roll:     roll,
	}
	return p.Serialize()
}
