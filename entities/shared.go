package entities

import "github.com/leNicDev/retromc/constants"

type GetBlock func(x int32, y byte, z int32, dim int32) constants.WBlock
type GetEntity func(entityId int32) constants.Entity
type FindNearbyPlayer func() (int32, bool)
type GetEntities func() []constants.Entity
type SendHealth func(entityId int32, newHp int16)
type BroadcastHurtAnim func(entityId int32)