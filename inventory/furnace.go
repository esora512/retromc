package inventory

import (
	"fmt"
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

type FurnacePosition struct {
	X int32
	Y int32
	Z int32
}

type Furnace struct {
	Size     uint16
	Items    []Item
	Position FurnacePosition

	Progress     int
	FuelRemain   int
	FuelDuration int
	IsSmelting   bool
}

func (f *Furnace) Smelt() (int, int, int) {
	if IsSmeltable(f.Items[0].TypeId) && IsFuel(f.Items[1].TypeId) {
		if f.IsSmelting {
			f.Progress += 1
			f.FuelRemain -= 1
			f.FuelDuration += 1
			return f.Progress, f.FuelDuration, f.FuelRemain
		} else {
			f.IsSmelting = true
			f.Progress = 0
			f.FuelRemain = FuelBurnTime(f.Items[1].TypeId)
			f.FuelDuration = 0
			return 0, 0, 0
		}
	}
	f.IsSmelting = false
	return 0, 0, 0
}

func (f *Furnace) Output() (bool, Item) {
	if f.Progress >= 200 {
		outItem := SmeltsTo(f.Items[0].TypeId)
		f.IsSmelting = false
		return true, Item{TypeId: outItem, Count: 1, Metadata: 0}
	}
	return false, Item{}
}

func TickFurnaces(sendProgress func(progress, fuelDuration, fuelRemain int), setSlot func(item Item, slot int16)) {
	for _, furnace := range furnaceRegistry {
		prog, dur, remain := furnace.Smelt()
		sendProgress(prog, dur, remain)

		exists, outItem := furnace.Output()
		if exists {
			furnace.Items[0].Count -= 1
			if furnace.Items[0].Count <= 0 {
				furnace.Items[0] = EmptyItem()
			}
			setSlot(furnace.Items[0], 0)

			if furnace.Items[2].TypeId == outItem.TypeId {
				furnace.Items[2].Count += 1
				outItem = furnace.Items[2]
			}
			furnace.Items[2] = outItem
			setSlot(outItem, 2)
			var newFuel Item
			fuel := furnace.Items[1]
			fuel.Count -= 1
			if fuel.Count <= 0 {
				newFuel = EmptyItem()
			} else {
				newFuel = fuel
			}
			furnace.Items[1] = newFuel
			setSlot(newFuel, 1)
		}
	}
}

func (f *Furnace) ShiftSlot(slot int16) int16 {
	return slot + 6
}

func NewFurnace() *Furnace {
	inv := Furnace{
		Size:         FURNACE_SIZE,
		FuelRemain:   200,
		Progress:     0,
		FuelDuration: 0,
		Items:        make([]Item, FURNACE_SIZE),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0, 0)
	}
	return &inv
}

func (f *Furnace) SetPosition(x, y, z int32) {
	f.Position = FurnacePosition{x, y, z}
}

func (f *Furnace) PeekItem(slot int16) Item {
	return f.Items[slot]
}

func (f *Furnace) SetItem(slot int16, typeId int16, count byte, metadata byte) {
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

var furnaceRegistry map[string]*Furnace

func initFurnaceRegistry() {
	if furnaceRegistry == nil {
		furnaceRegistry = make(map[string]*Furnace)
	}
}

func GetFurnace(x, y, z int32) *Furnace {
	if furnaceRegistry == nil {
		furnaceRegistry = make(map[string]*Furnace)
	}

	key := furnaceKey(x, y, z)

	if furnace, ok := furnaceRegistry[key]; ok {
		return furnace
	}
	return nil
}

func furnaceKey(x, y, z int32) string {
	return fmt.Sprintf("%d:%d:%d", x, y, z)
}

func PlaceFurnace(x, y, z int32) bool {
	initFurnaceRegistry()

	key := furnaceKey(x, y, z)

	if _, ok := furnaceRegistry[key]; ok {
		return false
	}

	furnace := NewFurnace()
	furnace.SetPosition(x, y, z)
	furnaceRegistry[key] = furnace
	return true
}

func RemoveFurnace(x, y, z int32) {
	if furnaceRegistry == nil {
		return
	}
	key := furnaceKey(x, y, z)
	delete(furnaceRegistry, key)
}
