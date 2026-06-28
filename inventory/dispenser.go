package inventory

import (
	"fmt"
	"log"
)

const DISPENSER_SIZE = 9

type DispenserPosition struct {
	X int32
	Y int32
	Z int32
}

type Dispenser struct {
	Size     uint16
	Items    []Item
	Position DispenserPosition
}

func NewDispenser() *Dispenser {
	inv := Dispenser{
		Size:  DISPENSER_SIZE,
		Items: make([]Item, DISPENSER_SIZE),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0, 0)
	}
	return &inv
}

func (d *Dispenser) SetPosition(x, y, z int32) {
	d.Position = DispenserPosition{x, y, z}
}

func (d *Dispenser) PeekItem(slot int16) Item {
	return d.Items[slot]
}

func (d *Dispenser) SetItem(slot int16, typeId int16, count byte, metadata uint16) {
	d.Items[slot] = NewItem(typeId, count, metadata)
}

func (d *Dispenser) RemoveOne(slot int16) int16 {
	if d.Items[slot].TypeId == -1 {
		return -1
	}
	if d.Items[slot].Count <= 1 {
		d.Items[slot] = NewItem(-1, 0, 0)
	} else {
		d.Items[slot].Count--
	}
	return slot
}

func (d *Dispenser) AddCount(slot int16, amount byte) {
	d.Items[slot].Count += amount
}

func (d *Dispenser) SetEmpty(slot int16) {
	d.Items[slot] = NewItem(-1, 0, 0)
}

func (d *Dispenser) Print() {
	log.Println("=== Dispenser ===")
	for i, item := range d.Items {
		if item.TypeId == -1 {
			continue
		}
		log.Printf("  slot %2d | id %-5d | count %d", i, item.TypeId, item.Count)
	}
	log.Println("=================")
}

var dispenserRegistry map[string]*Dispenser

func GetDispenser(x, y, z int32) *Dispenser {
	if dispenserRegistry == nil {
		dispenserRegistry = make(map[string]*Dispenser)
	}

	key := dispenserKey(x, y, z)

	if dispenser, ok := dispenserRegistry[key]; ok {
		return dispenser
	}
	return nil
}

func dispenserKey(x, y, z int32) string {
	return fmt.Sprintf("%d:%d:%d", x, y, z)
}

func initDispenserRegistry() {
	if dispenserRegistry == nil {
		dispenserRegistry = make(map[string]*Dispenser)
	}
}

func PlaceDispenser(x, y, z int32) bool {
	initDispenserRegistry()

	key := dispenserKey(x, y, z)

	if _, ok := dispenserRegistry[key]; ok {
		return false
	}

	dispenser := NewDispenser()
	dispenser.SetPosition(x, y, z)
	dispenserRegistry[key] = dispenser
	return true
}

func RemoveDispenser(x, y, z int32) {
	if dispenserRegistry == nil {
		return
	}
	key := dispenserKey(x, y, z)
	delete(dispenserRegistry, key)
}

func (d *Dispenser) IsInDispenser(slot int16) bool {
	return slot >= 0 && slot < int16(d.Size)
}
