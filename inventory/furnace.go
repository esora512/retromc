package inventory

import (
	"fmt"
	"log"

	"github.com/leNicDev/retromc/constants"
)

const FURNACE_SIZE = 3

func IsSmeltable(typeId int16) bool {
	if typeId == constants.IronOre.Value || typeId == constants.GoldOre.Value || typeId == constants.Sand.Value || typeId == constants.Cobblestone.Value {
		return true
	}
	return false
}

func IsFuel(typeId int16) bool {
	return typeId == constants.Planks.Value
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
			return 0, 0, 0
		}
	}
	f.IsSmelting = false
	return 0, 0, 0
}

func (f *Furnace) Output() (bool, Item) {
	if f.Progress >= 200 {
		f.Progress = 0
		return true, Item{TypeId: constants.Diamond.Value, Count: 1, Metadata: 0}
	}
	return false, Item{}
}

func TickFurnaces(sendProgress func(progress, fuelDuration, fuelRemain int), setSlot func(item Item)) {
	for _, furnace := range furnaceRegistry {
		prog, dur, remain := furnace.Smelt()
		sendProgress(prog, dur, remain)

		exists, outItem := furnace.Output()
		if exists {
			setSlot(outItem)
		}
	}
}

func (f *Furnace) ShiftSlot(slot int16) int16 {
	return slot + 6
}

func NewFurnace() *Furnace {
	inv := Furnace{
		Size:  FURNACE_SIZE,
		Items: make([]Item, FURNACE_SIZE),
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
