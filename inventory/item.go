package inventory

import (
	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/packet"
)

type Item struct {
	TypeId   int16
	Count    byte
	Metadata uint16
}

func (item *Item) IsHoe() bool {
	return item.TypeId == constants.WoodenHoe.Value ||
		item.TypeId == constants.StoneHoe.Value ||
		item.TypeId == constants.IronHoe.Value ||
		item.TypeId == constants.DiamondHoe.Value ||
		item.TypeId == constants.GoldHoe.Value
}

func (item *Item) IsAxe() bool {
	return item.TypeId == constants.WoodenAxe.Value ||
		item.TypeId == constants.StoneAxe.Value ||
		item.TypeId == constants.IronAxe.Value ||
		item.TypeId == constants.DiamondAxe.Value ||
		item.TypeId == constants.GoldAxe.Value
}

func (item *Item) IsPickaxe() bool {
	return item.TypeId == constants.WoodenPickaxe.Value ||
		item.TypeId == constants.StonePickaxe.Value ||
		item.TypeId == constants.IronPickaxe.Value ||
		item.TypeId == constants.DiamondPickaxe.Value ||
		item.TypeId == constants.GoldPickaxe.Value
}

func (item *Item) IsBoneMeal() bool {
	return item.TypeId == constants.Dye.Value && item.Metadata == 15
}

func (item *Item) IsShovel() bool {
		return item.TypeId == constants.WoodenShovel.Value ||
		item.TypeId == constants.StoneShovel.Value ||
		item.TypeId == constants.IronShovel.Value ||
		item.TypeId == constants.DiamondShovel.Value ||
		item.TypeId == constants.GoldShovel.Value
}

func (item *Item) Serialize() []byte {
	writer := packet.NewPacketWriter()

	writer.WriteShort(uint16(item.TypeId))
	// Per protocol: count and damage/metadata are only present when the item is not empty (-1)
	if item.TypeId != -1 {
		writer.WriteByte(item.Count)
		writer.WriteShort(item.Metadata)
	}

	return writer.Bytes()
}

func NewItem(typeId int16, count byte, metadata uint16) Item {
	return Item{
		TypeId:   typeId,
		Count:    count,
		Metadata: metadata,
	}
}

var nonStackableItems = map[int16]bool{
	constants.BedItem.Value: true,

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

	constants.WaterBucket.Value: true,
	constants.LavaBucket.Value:  true,
	constants.Bucket.Value:      true,
	constants.MilkBucket.Value:  true,

	constants.Apple.Value:          true,
	constants.GoldenApple.Value:    true,
	constants.Bread.Value:          true,
	constants.Porkchop.Value:       true,
	constants.CookedPorkchop.Value: true,
	constants.Fish.Value:           true,
	constants.CookedFish.Value:     true,
	constants.MushroomStew.Value:   true,

	constants.CakeItem.Value:        true,
	constants.Saddle.Value:          true,
	constants.Minecart.Value:        true,
	constants.ChestMinecart.Value:   true,
	constants.FurnaceMinecart.Value: true,
	constants.Boat.Value:            true,
	constants.WoodenDoorItem.Value:  true,
	constants.IronDoorItem.Value:    true,
	constants.Sign.Value:            true,
	constants.Map.Value:             true,
	constants.Record13.Value:        true,
	constants.RecordCat.Value:       true,
	constants.FlintAndSteel.Value:   true,
	constants.Shears.Value:          true,
	constants.Bow.Value:             true,
	constants.FishingRod.Value:      true,
}

func IsStackable(typeId int16) bool {
	// armor (298-317) never stacks
	return !nonStackableItems[typeId] && (typeId < constants.LeatherCap.Value || typeId > constants.GoldBoots.Value)
}
