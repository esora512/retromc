package inventory

import (
	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/packet"
)

type Item struct {
	TypeId   int16
	Count    byte
	Metadata byte
}

func (item *Item) Serialize() []byte {
	writer := packet.NewPacketWriter()

	writer.WriteShort(uint16(item.TypeId))
	// Per protocol: count and damage/metadata are only present when the item is not empty (-1)
	if item.TypeId != -1 {
		writer.WriteByte(item.Count)
		writer.WriteShort(uint16(item.Metadata))
	}

	return writer.Bytes()
}

func NewItem(typeId int16, count byte, metadata byte) Item {
	return Item{
		TypeId:   typeId,
		Count:    count,
		Metadata: metadata,
	}
}

var nonStackableItems = map[int16]bool{
	constants.IronShovel.Value:  true,
	constants.IronPickaxe.Value: true,
	constants.IronAxe.Value:     true,
	constants.IronSword.Value:   true,
	constants.IronHoe.Value:     true,

	constants.WoodenSword.Value:   true,
	constants.WoodenShovel.Value:  true,
	constants.WoodenPickaxe.Value: true,
	constants.WoodenAxe.Value:     true,
	constants.WoodenHoe.Value:     true,

	constants.StoneSword.Value:   true,
	constants.StoneShovel.Value:  true,
	constants.StonePickaxe.Value: true,
	constants.StoneAxe.Value:     true,
	constants.StoneHoe.Value:     true,

	constants.DiamondSword.Value:   true,
	constants.DiamondShovel.Value:  true,
	constants.DiamondPickaxe.Value: true,
	constants.DiamondAxe.Value:     true,
	constants.DiamondHoe.Value:     true,

	constants.GoldSword.Value:   true,
	constants.GoldShovel.Value:  true,
	constants.GoldPickaxe.Value: true,
	constants.GoldAxe.Value:     true,
	constants.GoldHoe.Value:     true,
}

func IsStackable(typeId int16) bool {
	return !nonStackableItems[typeId]
}
