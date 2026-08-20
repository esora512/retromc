package entities

import "github.com/leNicDev/retromc/constants"

type GetBlock func(x int32, y byte, z int32, dim int32) constants.WBlock

type WorldShared interface {
	IsNight() bool
	FindNearbyPlayer(m *Mob) (int32, bool)
	SnapshotEntities() []constants.Entity
	GetBlock(x int32, y byte, z int32, dim int32) constants.WBlock
	GetEntity(entityId int32) (constants.Entity, bool)
	SendHealth(entityId int32, newHp int16)
	BroadcastPain(entityId int32)
}
