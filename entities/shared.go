package entities

import "github.com/leNicDev/retromc/constants"

type GetBlock func(x int32, y byte, z int32, dim int32) constants.WBlock
type GetEntityPos func(entityId int32) (bool, float64, float64, float64)
type FindNearbyPlayer func() (int32, bool)