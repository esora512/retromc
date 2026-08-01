package inventory

import (
	"log"

	"github.com/leNicDev/retromc/constants"
)

const FURNACE_SIZE = 3

func FuelBurnTime(fuel int16) int {
	if fuel == constants.Planks.Value || fuel == constants.Log.Value || fuel == constants.CraftingTable.Value || fuel == constants.WoodenDoor.Value || fuel == constants.WoodenStairs.Value {
		return 300
	}

	if fuel == constants.Stick.Value {
		return 100
	}

	if fuel == constants.Coal.Value {
		return 1600
	}

	if fuel == constants.LavaBucket.Value {
		return 20000
	}

	if fuel == constants.Sapling.Value {
		return 100
	}
	return 0
}

func SmeltsTo(smeltable int16) int16 {
	if smeltable == constants.IronOre.Value {
		return constants.Iron.Value
	}
	if smeltable == constants.GoldOre.Value {
		return constants.Gold.Value
	}

	if smeltable == constants.Sand.Value {
		return constants.Glass.Value
	}

	if smeltable == constants.Cobblestone.Value {
		return constants.Stone.Value
	}

	if smeltable == constants.Clay.Value {
		return constants.Brick.Value
	}

	if smeltable == constants.Log.Value {
		return constants.Coal.Value
	}
	return 0
}

func IsSmeltable(typeId int16) bool {
	return SmeltsTo(typeId) != 0
}

func IsFuel(typeId int16) bool {
	return FuelBurnTime(typeId) != 0
}

type Furnace struct {
	Size     uint16
	Items    []Item
	Position ContainerPosition

	Progress   int
	FuelRemain int
	MaxFuel    int
	IsSmelting bool
	IsBurning  bool
	Dim int32
}

func (f *Furnace) Smelt(setSlot func(item Item, slot int16)) (int, int, int) {
	if !f.IsBurning {
		if IsSmeltable(f.Items[0].TypeId) && IsFuel(f.Items[1].TypeId) {
			f.MaxFuel = FuelBurnTime(f.Items[1].TypeId)
			f.FuelRemain = f.MaxFuel

			if f.Items[1].Count <= 1 {
				f.Items[1] = NewItem(-1, 0, 0)
			} else {
				f.Items[1].Count--
			}
			setSlot(f.Items[1], 1)
			f.IsBurning = true
		} else {
			f.Progress = 0
			return 0, 0, 0
		}
	}

	f.FuelRemain--
	if f.FuelRemain <= 0 {
		f.IsBurning = false
		f.FuelRemain = 0
		f.MaxFuel = 0
	}

	if IsSmeltable(f.Items[0].TypeId) {
		f.Progress++
	} else {
		f.Progress = 0
	}

	return f.Progress, f.MaxFuel, f.FuelRemain
}

func (f *Furnace) Output() (bool, Item) {
	if f.Progress >= 200 {
		outItem := SmeltsTo(f.Items[0].TypeId)
		f.Progress = 0
		return true, Item{TypeId: outItem, Count: 1, Metadata: 0}
	}
	return false, Item{}
}

func TickFurnaces(furnaces []*Furnace, 
	sendProgress func(progress, fuelMax, fuelRemain int), 
	setSlot func(item Item, slot int16), 
	setBlock func(x, y, z int16, lit bool, dim int32)) {
	for _, furnace := range furnaces {
		prog, fMax, remain := furnace.Smelt(setSlot)
		sendProgress(prog, fMax, remain)
		if furnace.IsBurning {
			setBlock(int16(furnace.Position.X), int16(furnace.Position.Y), int16(furnace.Position.Z), true, furnace.Dim)
		} else {
			setBlock(int16(furnace.Position.X), int16(furnace.Position.Y), int16(furnace.Position.Z), false, furnace.Dim)
		}
		exists, outItem := furnace.Output()
		if exists {
			if furnace.Items[0].Count <= 1 {
				furnace.Items[0] = NewItem(-1, 0, 0)
				furnace.IsSmelting = false
			} else {
				furnace.Items[0].Count -= 1
			}
			setSlot(furnace.Items[0], 0)

			if furnace.Items[2].TypeId == outItem.TypeId && furnace.Items[2].TypeId != -1 {
				furnace.Items[2].Count += 1
				outItem = furnace.Items[2]
			}
			furnace.Items[2] = outItem
			setSlot(furnace.Items[2], 2)
		}
	}
}

func (f *Furnace) ShiftSlot(slot int16) int16 {
	return slot + 6
}

func NewFurnace() *Furnace {
	inv := Furnace{
		Size:       FURNACE_SIZE,
		FuelRemain: 0,
		Progress:   0,
		MaxFuel:    0,
		Items:      make([]Item, FURNACE_SIZE),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0, 0)
	}
	return &inv
}

func (f *Furnace) SetPosition(x, y, z int32) {
	f.Position = ContainerPosition{X: x, Y: y, Z: z}
}

func (f *Furnace) PeekItem(slot int16) Item {
	return f.Items[slot]
}

func (f *Furnace) SetItem(slot int16, typeId int16, count byte, metadata uint16) {
	f.Items[slot] = NewItem(typeId, count, metadata)
}

func (f *Furnace) RemoveOne(slot int16) int16 {
	if f.Items[slot].TypeId == -1 {
		return -1
	}
	if f.Items[slot].Count <= 1 {
		f.Items[slot] = NewItem(-1, 0, 0)
	} else {
		f.Items[slot].Count--
	}
	return slot
}

func (f *Furnace) AddCount(slot int16, amount byte) {
	f.Items[slot].Count += amount
}

func (f *Furnace) SetEmpty(slot int16) {
	f.Items[slot] = NewItem(-1, 0, 0)
}

func (f *Furnace) Print() {
	log.Println("=== Furnace ===")
	for i, item := range f.Items {
		if item.TypeId == -1 {
			continue
		}
		log.Printf("  slot %2d | id %-5d | count %d", i, item.TypeId, item.Count)
	}
	log.Println("=================")
}
