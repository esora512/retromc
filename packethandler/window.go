package packethandler

import (
	"log"

	"net"

	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleWindowClickInPacket(connection net.Conn, p packets.WindowClickInPacket, pl *player.Player) {
	if p.WindowId != 0 {
		return
	}
	log.Printf("Window click: %+v", p)

	//TODO: Cases left to handle
	// Preconditions:
	// - Left/Right
	// - Shift/No Shift,
	// - New/Existing Item
	// - Same Item/Not Same Item (If same item is a stack, it may be equivalent to Not Same item)
	// - Source is always contains an item, else noop
	// - Target can contain an item or be empty

	// - Shift + right click Move a single item count to hotbar / main inventory (Bugged, no packet is sent from client)

	switch {
	// Left click: Manual pick up from slot
	case p.ItemID > 0 && p.RightClick == 0 && !p.Shift && !pl.SelectedItem.Selected:
		log.Printf("Item %d held", p.ItemID)
		pl.Inventory.Hold(p.Slot)
		pl.SelectedItem.SetItem(p.GetItem(), p.Slot, p.ActionNumber, p.RightClick == 1)

	// Left click: Place item in slot that is already occupied
	case p.ItemID > 0 && p.RightClick == 0 && !p.Shift && pl.SelectedItem.Selected:
		if pl.SelectedItem.Item.TypeId == pl.Inventory.PeekItem(p.Slot).TypeId {
			if pl.Inventory.PeekItem(p.Slot).Count == player.MaxStack {
				return
			}
			log.Printf("Item %d placed", p.ItemID)
			pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
			pl.SelectedItem.Selected = false
		} else {
			log.Printf("Item %d placed & Item %d held", pl.SelectedItem.Item.TypeId, p.ItemID)
			pl.Inventory.Hold(p.Slot)
			pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
			pl.SelectedItem.SetItem(p.GetItem(), p.Slot, p.ActionNumber, p.RightClick == 1)
		}

	// Left click: Place item in empty slot
	case p.ItemID < 0 && p.RightClick == 0 && !p.Shift && pl.SelectedItem.Selected:
		if p.Slot == -999 {
			pl.Inventory.Drop(pl.SelectedItem.Slot)
			pl.SelectedItem.Selected = false
		} else {
			log.Printf("Item %d placed", p.ItemID)
			pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
			pl.SelectedItem.Selected = false
		}
	// Shift + left click: Move item to hotbar / main inventory
	case p.ItemID > 0 && p.RightClick == 0 && p.Shift && !pl.SelectedItem.Selected:
		sourceSlot := p.Slot
		// Place item from hotbar to main inventory
		if pl.Inventory.IsHotbarSlot(sourceSlot) {
			targetSlot := pl.Inventory.FindFirstNonStackSlotOfItemInMainInventory(pl.Inventory.PeekItem(sourceSlot).TypeId)
			if targetSlot != -1 {
				pl.Inventory.Place(pl.Inventory.PeekItem(sourceSlot), targetSlot)
				pl.Inventory.RemoveAllFromSlot(sourceSlot)
			} else {

				targetSlot := pl.Inventory.FindFirstEmptySlotinMainInventory()
				if targetSlot == -1 {
					log.Printf("No empty slot in main inventory")
					return
				}
				pl.Inventory.Move(sourceSlot, targetSlot)
			}
			// Place item from main inventory to hotbar
		} else {
			targetSlot := pl.Inventory.FindFirstNonStackSlotOfItemInHotbar(pl.Inventory.PeekItem(sourceSlot).TypeId)
			if targetSlot != -1 {
				pl.Inventory.Place(pl.Inventory.PeekItem(sourceSlot), targetSlot)
				pl.Inventory.RemoveAllFromSlot(sourceSlot)
			} else {
				targetSlot = pl.Inventory.FindFirstEmptySlotinHotbar()
				pl.Inventory.Move(sourceSlot, targetSlot)
			}
			if targetSlot == -1 {
				log.Printf("No empty slot in hotbar")
				return
			}
		}
	// Right click: Split item stack and hold
	case p.ItemID > 0 && p.RightClick == 1 && !p.Shift && !pl.SelectedItem.Selected:
		if pl.Inventory.PeekItem(p.Slot).Count > 1 {
			pl.Inventory.HoldHalf(p.Slot)
			log.Printf("Item %d held", p.ItemID)
			item := p.GetItem()
			item.Half()
			pl.SelectedItem.SetItem(item, p.Slot, p.ActionNumber, p.RightClick == 1)
		} else {
			pl.Inventory.Hold(p.Slot)
			log.Printf("Item %d held", p.ItemID)
			pl.SelectedItem.SetItem(p.GetItem(), p.Slot, p.ActionNumber, p.RightClick == 1)
		}

	case p.ItemID > 0 && p.RightClick == 1 && !p.Shift && pl.SelectedItem.Selected:
		if pl.SelectedItem.Item.TypeId == pl.Inventory.PeekItem(p.Slot).TypeId && pl.Inventory.PeekItem(p.Slot).Count < player.MaxStack {
			pl.Inventory.PlaceOne(&pl.SelectedItem.Item, p.Slot)
		} else {
			log.Printf("Item %d placed & Item %d held", pl.SelectedItem.Item.TypeId, p.ItemID)
			pl.Inventory.Hold(p.Slot)
			pl.Inventory.Place(pl.SelectedItem.Item, p.Slot)
			pl.SelectedItem.SetItem(p.GetItem(), p.Slot, p.ActionNumber, p.RightClick == 1)
		}
	case p.ItemID < 0 && p.RightClick == 1 && !p.Shift && pl.SelectedItem.Selected:
		if p.Slot == -999 {
			pl.SelectedItem.Item.Count--
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
		} else if pl.Inventory.PeekItem(p.Slot).TypeId == -1 {
			pl.Inventory.PlaceOneInEmpty(&pl.SelectedItem.Item, p.Slot)
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
		}
	}
	transactionOutPacket := packets.TransactionOutPacket{
		WindowId:     0,
		ActionNumber: p.ActionNumber,
		Accepted:     true,
	}
	connection.Write(transactionOutPacket.Serialize())
	pl.SelectedItem.Print()
	pl.Inventory.Print()
}
