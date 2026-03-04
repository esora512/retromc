package packethandler

import (
	"log"

	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleWindowClickInPacket(p packets.WindowClickInPacket, pl *player.Player) {
	if p.WindowId != 0 {
		return
	}
	log.Printf("Window click: %+v", p)

	//TODO: Cases left to handle
	// - Right click: Place item in slot that is already occupied
	// - Shift + left click when item is moved to a non-full stack in hotbar / main inventory
	// - Shift + right click: Move a single item count to hotbar / main inventory

	switch {
	// Left click: Manual pick up from slot
	case p.ItemID > 0 && p.RightClick == 0 && !p.Shift && !pl.SelectedItem.Selected:
		pl.Inventory.Hold(p.Slot)
		log.Printf("Item %d held", p.ItemID)
		pl.SelectedItem = player.SelectedItem{
			Item: player.Item{
				TypeId: p.ItemID,
				Count:  p.ItemCount,
				Uses:   p.ItemUses,
			},
			Selected:     true,
			ActionNumber: p.ActionNumber,
			Slot:         p.Slot,
			RightClick:   false,
		}
	// Left click: Place item in slot that is already occupied
	case p.ItemID > 0 && p.RightClick == 0 && !p.Shift && pl.SelectedItem.Selected && !pl.SelectedItem.RightClick:
		if pl.SelectedItem.Item.TypeId == pl.Inventory.PeekItem(p.Slot).TypeId {
			log.Printf("Item %d placed", p.ItemID)
			pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
		} else {
			log.Printf("Item %d placed & Item %d held", pl.SelectedItem.Item.TypeId, p.ItemID)
			pl.Inventory.Hold(p.Slot)
			pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
			pl.SelectedItem = player.SelectedItem{
				Item: player.Item{
					TypeId: p.ItemID,
					Count:  p.ItemCount,
					Uses:   p.ItemUses,
				},
				Selected:     true,
				ActionNumber: p.ActionNumber,
				Slot:         p.Slot,
				RightClick:   false,
			}

		}
	// Left click: Place item in empty slot
	case p.ItemID < 0 && p.RightClick == 0 && !p.Shift && pl.SelectedItem.Selected:
		log.Printf("Item %d placed", p.ItemID)
		pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
		pl.SelectedItem.Selected = false
	// Shift + left click: Move item to hotbar / main inventory
	case p.ItemID > 0 && p.RightClick == 0 && p.Shift && !pl.SelectedItem.Selected:
		sourceSlot := p.Slot
		if pl.Inventory.IsHotbarSlot(sourceSlot) {
			targetSlot := pl.Inventory.FindFirstEmptySlotinMainInventory()
			if targetSlot == -1 {
				log.Printf("No empty slot in main inventory")
				return
			}
			pl.Inventory.Move(sourceSlot, targetSlot)
		} else {
			targetSlot := pl.Inventory.FindFirstEmptySlotinHotbar()
			if targetSlot == -1 {
				log.Printf("No empty slot in hotbar")
				return
			}
			pl.Inventory.Move(sourceSlot, targetSlot)
		}
	// Right click: Split item stack and hold
	case p.ItemID > 0 && p.RightClick == 1 && !p.Shift && !pl.SelectedItem.Selected:
		pl.Inventory.HoldHalf(p.Slot)
		log.Printf("Item %d held", p.ItemID)
		pl.SelectedItem = player.SelectedItem{
			Item: player.Item{
				TypeId: p.ItemID,
				Count:  p.ItemCount / 2,
				Uses:   p.ItemUses,
			},
			Selected:     true,
			ActionNumber: p.ActionNumber,
			Slot:         p.Slot,
			RightClick:   true,
		}
	}

	pl.Inventory.Print()
}
