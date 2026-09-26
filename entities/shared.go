package entities

import (
	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

type WorldShared interface {
	NewInteractWithBlockPacket(pl *player.Player, bedType byte) []byte
	NewEntityMetadataPacket(e constants.Entity, m []byte) []byte
	IsNight() bool
	IsLoaded(x, z, dim int32) bool
	FindNearbyPlayer(m *Mob) (int32, bool)
	SnapshotEntities() []constants.Entity
	GetBlock(x int32, y byte, z int32, dim int32) constants.WBlock
	SetBlockInQueue(x, y, z int32, block constants.WBlock, dim int32)
	NotifyBlockRemoved(x, y, z int32, oldType byte, dim int32)
	GetEntity(entityId int32) (constants.Entity, bool)
	SendHealth(entityId int32, newHp int16)
	HurtPlayer(pl *player.Player, attacker constants.Entity, dmg int16) int16
	DropItemFromMinedBlock(x, y, z float64, blockItem int16, blockMeta byte, count byte, dim, delay int32)

	BroadcastPacket(data []byte)

	SpawnPlayerPacket(target constants.Entity) []byte
	SpawnObjectPacket(target constants.Entity) []byte
	SpawnMobPacket(target constants.Entity) []byte
	SpawnItemPacket(target constants.Entity) []byte

	NewCollectItemPacket(itemId, collectorId int32) []byte

	NewAnimationPacket(pl *player.Player, anim byte) []byte

	NewEntityEventPacket(e constants.Entity, action byte) []byte

	NewPositionAndRotationOrTeleportPacket(
		target constants.Entity,
		state constants.MovementState,
	) []byte

	NewPositionPacket(
		target constants.Entity,
		state constants.MovementState,
	) []byte

	NewMobPositionAndRotationOrTeleportPacket(
		target constants.Entity,
		state constants.MovementState,
	) []byte

	NewEntityVelocityPacket(
		entityID int32,
		state constants.MovementState,
	) []byte

	NewRotationPacket(
		target constants.Entity,
		state constants.MovementState,
	) []byte

	NewTeleportPacket(
		target constants.Entity,
		state constants.MovementState,
	) []byte

	SetEquipment(pl, viewer *player.Player)

	DespawnEntity(eId int32) []byte

	GetPlayers() map[int32]*player.Player

	GetPlayer(pId int32) (*player.Player, bool)

	RemoveEntity(eId int32)
}
