package packets

import (
	"math"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type TeleportEntityPacket struct {
	EntityId int32
	X        int32
	Y        int32
	Z        int32
	Yaw      byte
	Pitch    byte
}

type AddPassengerPacket struct {
	Passenger int32
	Vehicle   int32
}

type SpawnObjectPacket struct {
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

type SpawnItemPacket struct {
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

type EntityPositionAndRotationPacket struct {
	EntityId int32
	X        byte
	Y        byte
	Z        byte
	Yaw      byte
	Pitch    byte
}

type EntityPositionPacket struct {
	EntityId int32
	X        byte
	Y        byte
	Z        byte
}

type EntityRotationPacket struct {
	EntityId int32
	Yaw      byte
	Pitch    byte
}

type DespawnEntityPacket struct {
	EntityId int32
}

type EntityEventPacket struct {
	EntityId int32
	Action   byte
}

type CollectItemPacket struct {
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

func (p *CollectItemPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.CollectItem)
	w.WriteInt32(p.ItemId)
	w.WriteInt32(p.CollectorId)
	return w.Bytes()
}

func (p *SpawnItemPacket) Serialize() []byte {
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

type EntityVelocityPacket struct {
	EntityId   int32
	Vx, Vy, Vz float64
}

func (p *EntityVelocityPacket) Serialize() []byte {
	vx := int16(clamp(p.Vx, -3.9, 3.9) * 8000)
	vy := int16(clamp(p.Vy, -3.9, 3.9) * 8000)
	vz := int16(clamp(p.Vz, -3.9, 3.9) * 8000)

	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityVelocity)
	writer.WriteInt32(p.EntityId)
	writer.WriteShort(uint16(vx))
	writer.WriteShort(uint16(vy))
	writer.WriteShort(uint16(vz))
	return writer.Bytes()
}

func (p *EntityPositionAndRotationPacket) Serialize() []byte {
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

func (p *EntityPositionPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityPosition)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.X)
	writer.WriteByte(p.Y)
	writer.WriteByte(p.Z)
	return writer.Bytes()
}

func (p *EntityRotationPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.EntityRotation)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.Yaw)
	writer.WriteByte(p.Pitch)
	return writer.Bytes()
}

func (p *DespawnEntityPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.DespawnEntity)
	writer.WriteInt32(p.EntityId)
	return writer.Bytes()
}

func (p *SpawnObjectPacket) Serialize() []byte {
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

func (p *AddPassengerPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.AddPassenger)
	writer.WriteInt32(p.Passenger)
	writer.WriteInt32(p.Vehicle)
	return writer.Bytes()
}

func (p *TeleportEntityPacket) Serialize() []byte {
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

func NewAddPassengerPacket(passenger, vehicle int32) []byte {
	p := AddPassengerPacket{
		Passenger: passenger,
		Vehicle:   vehicle,
	}
	return p.Serialize()
}

func NewEntityDespawnPacket(id int32) []byte {
	p := DespawnEntityPacket{
		EntityId: id,
	}
	return p.Serialize()
}

func NewTeleportPlayerPacket(pl *player.Player, x, y, z, yaw, pitch float64, world *level.World) []byte {
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))
	p := TeleportEntityPacket{
		EntityId: pl.GetEntityId(),
		X:        encX,
		Y:        encY,
		Z:        encZ,
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

const maxRelDelta = 127

func NewMobPositionAndRotationPacket(m *level.Mob, x, y, z, yaw, pitch float64) []byte {
	//log.Printf("Spider Pos&Rot Pkt x=%f, y=%f, z=%f (Id=%d)", x, y, z, m.EntityId)
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))

	dX := encX - int32(math.Floor(m.X*32))
	dY := encY - int32(math.Floor(m.Y*32))
	dZ := encZ - int32(math.Floor(m.Z*32))
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))

	if dX < -maxRelDelta || dX > maxRelDelta ||
		dY < -maxRelDelta || dY > maxRelDelta ||
		dZ < -maxRelDelta || dZ > maxRelDelta {

		p := TeleportEntityPacket{
			EntityId: m.EntityId,
			X:        encX,
			Y:        encY,
			Z:        encZ,
			Yaw:      byte(dYaw),
			Pitch:    byte(dPitch),
		}
		return p.Serialize()
	}

	p := EntityPositionAndRotationPacket{
		EntityId: m.EntityId,
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func BroadcastMobPositionAndRotation(w *level.World, m *level.Mob, nX, nY, nZ, yaw, pitch float64) {
	p := NewMobPositionAndRotationPacket(m, nX, nY, nZ, yaw, pitch)
	w.BroadcastPacket(p)
}

func NewPlayerPositionAndRotationPacket(pl *player.Player, x, y, z, yaw, pitch float64) []byte {
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))

	dX := encX - int32(math.Floor(pl.X*32))
	dY := encY - int32(math.Floor(pl.Y*32))
	dZ := encZ - int32(math.Floor(pl.Z*32))
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))

	if dX < -maxRelDelta || dX > maxRelDelta ||
		dY < -maxRelDelta || dY > maxRelDelta ||
		dZ < -maxRelDelta || dZ > maxRelDelta {

		// log.Printf(
		// 	"[PosLook] entity=%d delta too large (dX=%d dY=%d dZ=%d), sending Teleport",
		// 	pl.EntityId, dX, dY, dZ,
		// )

		p := TeleportEntityPacket{
			EntityId: int32(pl.EntityId),
			X:        encX,
			Y:        encY,
			Z:        encZ,
			Yaw:      byte(dYaw),
			Pitch:    byte(dPitch),
		}
		return p.Serialize()
	}

	p := EntityPositionAndRotationPacket{
		EntityId: int32(pl.EntityId),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func NewPlayerPositionPacket(pl *player.Player, x, y, z float64, world *level.World) []byte {
	encX := int32(math.Floor(x * 32))
	encY := int32(math.Floor(y * 32))
	encZ := int32(math.Floor(z * 32))

	dX := encX - int32(math.Floor(pl.X*32))
	dY := encY - int32(math.Floor(pl.Y*32))
	dZ := encZ - int32(math.Floor(pl.Z*32))

	if dX < -maxRelDelta || dX > maxRelDelta ||
		dY < -maxRelDelta || dY > maxRelDelta ||
		dZ < -maxRelDelta || dZ > maxRelDelta {

		p := TeleportEntityPacket{
			EntityId: int32(pl.EntityId),
			X:        encX,
			Y:        encY,
			Z:        encZ,
			Yaw:      byte(0),
			Pitch:    byte(0),
		}
		return p.Serialize()
	}

	p := EntityPositionPacket{
		EntityId: int32(pl.EntityId),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	return p.Serialize()
}

func NewPlayerRotationPacket(pl *player.Player, yaw, pitch float64, world *level.World) []byte {
	dYaw := int32(math.Floor(yaw * 256 / 360))
	dPitch := int32(math.Floor(pitch * 256 / 360))

	p := EntityRotationPacket{
		EntityId: int32(pl.EntityId),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

type PlayerActionPacket struct {
	packet.Packet
	EntityId int
	ActionId byte
}

func ReadPlayerActionPacket(reader *packet.PacketReader) PlayerActionPacket {
	packet := PlayerActionPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.EntityId = reader.ReadInt()
	packet.ActionId = reader.ReadByte()
	//log.Printf("Entity action: %+v", packet)
	return packet
}

type InteractWithEntityPacket struct {
	packet.Packet
	EntityId int32
	PlayerId int32
	Attack   bool // true = left click, false = right click
}

func ReadInteractWithEntityPacket(reader *packet.PacketReader) InteractWithEntityPacket {
	packet := InteractWithEntityPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.PlayerId = reader.ReadInt32()
	packet.EntityId = reader.ReadInt32()
	packet.Attack = reader.ReadBool()
	return packet
}

type EntityMetadataPacket struct {
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

func (p *EntityMetadataPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.EntityMetadata)
	w.WriteInt32(p.EntityId)
	w.Write(p.Metadata)
	return w.Bytes()
}

func NewPlayerMetadataPacketSneak(pl *player.Player, sneaking bool) []byte {
	p := EntityMetadataPacket{
		EntityId: int32(pl.EntityId),
		Metadata: sneakMetadata(sneaking),
	}
	return p.Serialize()
}

func PlayerEntityMetadataPacketRiding(pl *player.Player, riding bool) []byte {
	p := EntityMetadataPacket{
		EntityId: int32(pl.EntityId),
		Metadata: ridingMetadata(riding),
	}
	return p.Serialize()
}

func (p *EntityEventPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.EntityEvent)
	w.WriteInt32(p.EntityId)
	w.WriteByte(p.Action)
	return w.Bytes()
}

func CollectItem(itemId, collectorId int32) []byte {
	p := CollectItemPacket{ItemId: itemId, CollectorId: collectorId}
	return p.Serialize()
}

func NewSpawnDroppedItem(w *level.World, itemId int16, amount, meta byte, x, y, z int32, yaw, pitch, roll byte, pickupDelay int32, dim int32) []byte {
	entityId := w.AddDroppedItem(x, y, z, int32(itemId), amount, meta, pickupDelay, dim)
	p := SpawnItemPacket{
		EntityId: entityId,
		ItemId:   itemId,
		Amount:   amount,
		Metadata: meta,
		X:        x*32 + 16,
		Y:        y*32 + 16,
		Z:        z*32 + 16,
		Yaw:      yaw,
		Pitch:    pitch,
		Roll:     roll,
	}
	return p.Serialize()
}

type SpawnMobPacket struct {
	EntityId int32
	MobType  byte
	Metadata byte
	X        int32
	Y        int32
	Z        int32
	Yaw      byte
	Pitch    byte
}

func (p *SpawnMobPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SpawnMob)
	w.WriteInt32(p.EntityId)
	w.WriteByte(p.MobType)
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteByte(p.Yaw)
	w.WriteByte(p.Pitch)
	w.WriteByte(0x7f)
	return w.Bytes()
}

func NewSpawnMob(w *level.World, mobType, meta byte, x, y, z int32, yaw, pitch byte, dim int32, entityId int32) []byte {
	p := SpawnMobPacket{
		EntityId: entityId,
		MobType:  mobType,
		Metadata: meta,
		X:        x*32 + 16,
		Y:        y*32 + 16,
		Z:        z*32 + 16,
		Yaw:      yaw,
		Pitch:    pitch,
	}
	return p.Serialize()
}

func BroadcastMobSpawn(w *level.World, mobType, meta byte, x, y, z int32, yaw, pitch byte, dim int32, entityId int32) {
	p := NewSpawnMob(w, mobType, meta, x, y, z, yaw, pitch, dim, entityId)
	w.BroadcastPacket(p)
}
