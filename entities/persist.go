package entities

import (
	"github.com/leNicDev/retromc/constants"
	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/mcregion"
)

var mobNames = map[byte]string{
	c.Creeper: "Creeper", c.Skeleton: "Skeleton", c.Spider: "Spider",
	c.Zombie: "Zombie", c.Pig: "Pig", c.Sheep: "Sheep",
}

func baseNBT(id string, x, y, z, vx, vy, vz float64, yaw, pitch, fall float32, fire, air int16, onGround bool) *mcregion.Compound {
	n := mcregion.NewCompound()
	n.String("id", id)
	n.DoubleList("Pos", []float64{x, y, z})
	n.DoubleList("Motion", []float64{vx, vy, vz})
	n.FloatList("Rotation", []float32{yaw, pitch})
	n.Float("FallDistance", fall)
	n.Short("Fire", fire)
	n.Short("Air", air)
	var g byte
	if onGround {
		g = 1
	}
	n.Byte("OnGround", g)
	return n
}

// EntityToNBT returns nil for entities that aren't persisted (players, falling blocks, dead ones)
func EntityToNBT(e constants.Entity) *mcregion.Compound {
	switch v := e.(type) {
	case *Mob:
		name, ok := mobNames[v.MobType]
		if !ok || v.HP <= 0 {
			return nil
		}
		n := baseNBT(name, v.X, v.Y, v.Z, v.Vx, v.Vy, v.Vz, v.RotYaw, v.RotPitch, float32(v.fallDistance), int16(v.FireTicks), int16(v.air), v.OnGround)
		n.Short("Health", v.HP)
		n.Short("HurtTime", int16(v.hurtTime))
		n.Short("DeathTime", 0)
		n.Short("AttackTime", int16(v.AttackCooldown))
		switch v.MobType {
		case c.Sheep:
			n.Byte("Color", v.Color)
			var sheared byte
			if v.Sheared {
				sheared = 1
			}
			n.Byte("Sheared", sheared)
		case c.Pig:
			n.Byte("Saddle", 0)
		}
		return n
	case *DroppedItem:
		if v.Dead || v.CollectorId != -1 || v.Amount == 0 {
			return nil
		}
		n := baseNBT("Item", v.X, v.Y, v.Z, v.VelX, v.VelY, v.VelZ, 0, 0, 0, 0, 300, v.OnGround)
		n.Short("Health", int16(itemMaxHealth-v.damage))
		n.Short("Age", int16(v.Age))
		item := mcregion.NewCompound()
		item.Short("id", int16(v.ItemId))
		item.Short("Damage", int16(v.Metadata))
		item.Byte("Count", v.Amount)
		n.AddCompound("Item", item)
		return n
	case *RideableEntity:
		if v.HP <= 0 || v.ShouldDespawn {
			return nil
		}
		id := "Boat"
		if v.ObjectType == c.ObjectMinecart {
			id = "Minecart"
		}
		n := baseNBT(id, v.X, v.Y, v.Z, v.VelocityX, v.VelocityY, v.VelocityZ, float32(v.YawDegrees), 0, 0, 0, 300, true)
		if v.ObjectType == c.ObjectMinecart {
			n.Int("Type", 0)
		}
		return n
	}
	return nil
}

func listDoubles(t *mcregion.Tag, name string) []float64 {
	l := t.Get(name)
	if l == nil {
		return nil
	}
	out := make([]float64, 0, len(l.List))
	for _, e := range l.List {
		out = append(out, e.DoubleVal)
	}
	return out
}

func listFloats(t *mcregion.Tag, name string) []float32 {
	l := t.Get(name)
	if l == nil {
		return nil
	}
	out := make([]float32, 0, len(l.List))
	for _, e := range l.List {
		out = append(out, e.FloatVal)
	}
	return out
}

func tagShort(t *mcregion.Tag, name string, def int16) int16 {
	if v := t.Get(name); v != nil {
		return v.ShortVal
	}
	return def
}

func tagByte(t *mcregion.Tag, name string) byte {
	if v := t.Get(name); v != nil {
		return v.ByteVal
	}
	return 0
}

// EntityFromNBT recreates a saved entity with a fresh id, or returns nil for unknown ones
func EntityFromNBT(t *mcregion.Tag, id int32, dim int32) constants.Entity {
	idTag := t.Get("id")
	pos := listDoubles(t, "Pos")
	if idTag == nil || len(pos) < 3 {
		return nil
	}
	motion := listDoubles(t, "Motion")
	if len(motion) < 3 {
		motion = []float64{0, 0, 0}
	}
	rot := listFloats(t, "Rotation")
	if len(rot) < 2 {
		rot = []float32{0, 0}
	}
	onGround := tagByte(t, "OnGround") != 0

	switch idTag.StrVal {
	case "Item":
		item := t.Get("Item")
		if item == nil {
			return nil
		}
		d := &DroppedItem{
			EntityId:    id,
			ItemId:      int32(tagShort(item, "id", 0)),
			Amount:      tagByte(item, "Count"),
			Metadata:    uint16(tagShort(item, "Damage", 0)),
			X:           pos[0],
			Y:           pos[1],
			Z:           pos[2],
			VelX:        motion[0],
			VelY:        motion[1],
			VelZ:        motion[2],
			Dim:         dim,
			DespawnIn:   -1,
			CollectorId: -1,
			Age:         int(tagShort(t, "Age", 0)),
			OnGround:    onGround,
			damage:      itemMaxHealth - int(tagShort(t, "Health", itemMaxHealth)),
		}
		if d.ItemId <= 0 || d.Amount == 0 {
			return nil
		}
		d.InitSyncState()
		return d
	case "Boat", "Minecart":
		objType := c.ObjectBoat
		if idTag.StrVal == "Minecart" {
			objType = c.ObjectMinecart
		}
		return &RideableEntity{
			EntityId:   id,
			X:          pos[0],
			Y:          pos[1],
			Z:          pos[2],
			VelocityX:  motion[0],
			VelocityY:  motion[1],
			VelocityZ:  motion[2],
			YawDegrees: float64(rot[0]),
			ObjectType: objType,
			HP:         4,
			Dimension:  dim,
		}
	}

	for mobType, name := range mobNames {
		if name != idTag.StrVal {
			continue
		}
		m := NewMob(id, mobType, pos[0], pos[1], pos[2], dim)
		m.Vx, m.Vy, m.Vz = motion[0], motion[1], motion[2]
		m.RotYaw, m.RotPitch = rot[0], rot[1]
		m.OnGround = onGround
		if f := t.Get("FallDistance"); f != nil {
			m.fallDistance = float64(f.FloatVal)
		}
		m.FireTicks = int32(tagShort(t, "Fire", 0))
		m.air = int32(tagShort(t, "Air", mobMaxAir))
		m.HP = tagShort(t, "Health", MaxHealth(mobType))
		m.hurtTime = int32(tagShort(t, "HurtTime", 0))
		m.AttackCooldown = int32(tagShort(t, "AttackTime", 0))
		if mobType == c.Sheep {
			m.Color = tagByte(t, "Color") & 0x0F
			m.Sheared = tagByte(t, "Sheared") != 0
		}
		if m.HP <= 0 {
			return nil
		}
		m.syncState()
		return m
	}
	return nil
}
