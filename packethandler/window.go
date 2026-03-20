package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/crafting"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleWorkbench(p packets.WindowClickInPacket, pl *player.Player, shift, rightClick bool) {
	targetSlot := p.Slot
	if targetSlot == 0 {
		craftInWorkbench(pl, shift, rightClick)
	} else {
		workbenchGridClick(pl, targetSlot, rightClick, shift)
	}
}

// workbenchGridClick handles click mechanics for workbench grid slots (1-9),
// mirroring the same pick/place/swap/merge logic as normalClick for inventory.
func workbenchGridClick(pl *player.Player, slot int16, rightClick, shift bool) {
	gridItem := pl.Workbench.GetGridItem(slot)
	heldItem := pl.SelectedItem.Item
	hasHeld := pl.SelectedItem.Selected
	slotEmpty := gridItem.TypeId == -1

	switch {
	// nothing in slot and nothing held
	case slotEmpty && !hasHeld:
		// noop

	// nothing in slot and something held
	case slotEmpty && hasHeld:
		if rightClick {
			// Place one item from held into the empty grid slot.
			pl.Workbench.SetGridItem(slot, inventory.NewItem(heldItem.TypeId, 1, heldItem.Metadata))
			pl.SelectedItem.Item.Count--
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
		} else {
			// Place entire held stack into the empty grid slot.
			pl.Workbench.SetGridItem(slot, inventory.NewItem(heldItem.TypeId, heldItem.Count, heldItem.Metadata))
			pl.SelectedItem.Selected = false
		}

	// something in slot, nothing held
	case !slotEmpty && !hasHeld:
		if rightClick {
			// Pick up half (rounded up) from grid slot.
			pickCount := (gridItem.Count + 1) / 2
			remainCount := gridItem.Count - pickCount
			picked := inventory.NewItem(gridItem.TypeId, pickCount, gridItem.Metadata)
			if remainCount <= 0 {
				pl.Workbench.SetGridItem(slot, inventory.EmptyItem())
			} else {
				pl.Workbench.SetGridItem(slot, inventory.NewItem(gridItem.TypeId, remainCount, gridItem.Metadata))
			}
			pl.SelectedItem.SetItem(picked, slot, 0, true)
		} else {
			// Pick up entire stack from grid slot.
			picked := inventory.NewItem(gridItem.TypeId, gridItem.Count, gridItem.Metadata)
			pl.Workbench.SetGridItem(slot, inventory.EmptyItem())
			pl.SelectedItem.SetItem(picked, slot, 0, false)
		}

	// something in slot, something held
	case !slotEmpty && hasHeld:
		sameType := gridItem.TypeId == heldItem.TypeId && gridItem.Metadata == heldItem.Metadata

		if sameType && inventory.IsStackable(heldItem.TypeId) {
			// Merge held into grid slot (up to MaxStack).
			addCount := heldItem.Count
			if rightClick {
				addCount = 1
			}
			room := inventory.MaxStack - int(gridItem.Count)
			if int(addCount) > room {
				addCount = byte(room)
			}
			if addCount == 0 {
				return
			}
			pl.SelectedItem.Item.Count -= addCount
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
			gridPtr := pl.Workbench.GetGridItemPtr(slot)
			if gridPtr != nil {
				gridPtr.Count += addCount
			}
		} else {

			// Different types: swap grid slot and held.
			if int(heldItem.Count) <= inventory.MaxStack {
				swapped := inventory.NewItem(gridItem.TypeId, gridItem.Count, gridItem.Metadata)
				pl.Workbench.SetGridItem(slot, inventory.NewItem(heldItem.TypeId, heldItem.Count, heldItem.Metadata))
				pl.SelectedItem.SetItem(swapped, slot, 0, rightClick)
			}
		}
	}
	result := crafting.Craft3x3(pl.Workbench.GetGrid())
	if result.TypeId != 1 {
		sendSetSlot(pl.Connection, 1, 0, inventory.NewItem(result.TypeId, result.Count, result.Metadata))
	}

}

// Refer to method: func_27085_a in Container.java in decompiled Minecraft Beta 1.7.3 server
// Refer to method: func_20007_a in NetServerHandler.java in decompiled Minecraft Beta 1.7.3 server
func handleWindowClickInPacket(connection net.Conn, p packets.WindowClickInPacket, pl *player.Player) {
	log.Println("Window click:", p.WindowId)
	p.Print()

	rightClick := p.RightClick == 1
	shift := p.Shift
	slot := p.Slot
	windowId := p.WindowId

	if slot == -999 {
		// Outside-inventory click: drop held stack / single item.
		// TODO: Impelment proper dropping in world
		if pl.SelectedItem.Selected {
			if rightClick {
				// Drop one item from the held stack.
				pl.SelectedItem.Item.Count--
				if pl.SelectedItem.Item.Count == 0 {
					pl.SelectedItem.Selected = false
				}
			} else {
				// Drop entire held stack.
				pl.SelectedItem.Selected = false
			}
		}
		acceptTransaction(connection, p)
		return
	}

	// Workbench actions
	if windowId == 1 {
		if slot > 9 {
			// If outside crafting grid, switch to inventory
			windowId = 0
			slot -= 1
		} else {
			if shift && slot != 0 {
				shiftClickWorkbench(pl, slot)
			} else {
				handleWorkbench(p, pl, shift, rightClick)
			}
			acceptTransaction(connection, p)
			pl.Workbench.Print()
			return
		}
	}

	// Regular inventory actions
	// Includes 2x2 grid crafting
	if windowId == 0 {
		if slot == 0 {
			craftInInventory(pl, shift, rightClick)
			acceptTransaction(connection, p)
			return
		}

		if shift {
			// Based on decompiled Beta 1.7.3 code, shift click is the same for right & left
			shiftClick(pl, slot)
		} else {
			normalClick(pl, slot, rightClick)
		}

		acceptTransaction(connection, p)
		pl.SelectedItem.Print()
		pl.Inventory.Print()
	}
}

// Covers all non-shift left- and right-click cases.
func normalClick(pl *player.Player, slot int16, rightClick bool) {
	slotItem := pl.Inventory.PeekItem(slot)
	heldItem := pl.SelectedItem.Item
	hasHeld := pl.SelectedItem.Selected
	slotEmpty := slotItem.TypeId == -1

	switch {
	// nothing in slot and nothing held
	case slotEmpty && !hasHeld:
		// noop

	// nothing in slot and something held
	case slotEmpty && hasHeld:
		if rightClick {
			// Place one item into the empty slot.
			pl.Inventory.PlaceOneInEmpty(&pl.SelectedItem.Item, slot)
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
		} else {
			// Place entire held stack into the empty slot.
			pl.Inventory.Place(heldItem, slot)
			pl.SelectedItem.Selected = false
		}

	// something in slot, nothing held
	case !slotEmpty && !hasHeld:
		if rightClick {
			// Decompiled code: var10 = mouseClick == 0 ? var13.stackSize : (var13.stackSize + 1) / 2;
			// For right click, we get (n + 1 / 2)
			pickCount := (slotItem.Count + 1) / 2
			picked := pl.Inventory.RemoveCountFromInventory(slot, pickCount)
			pl.SelectedItem.SetItem(picked, slot, 0, true)
		} else {
			picked := pl.Inventory.Hold(slot)
			pl.SelectedItem.SetItem(picked, slot, 0, false)
		}

	// something in slot, something held
	case !slotEmpty && hasHeld:
		sameType := slotItem.TypeId == heldItem.TypeId && slotItem.Metadata == heldItem.Metadata

		if sameType && inventory.IsStackable(heldItem.TypeId) {
			// Merge held into slot (up to MaxStack).
			addCount := heldItem.Count
			if rightClick {
				addCount = 1
			}
			// Cap by slot limit and item max stack.
			room := inventory.MaxStack - int(slotItem.Count)
			if int(addCount) > room {
				addCount = byte(room)
			}
			if addCount == 0 {
				return
			}
			// Deduct from held and add to slot.
			pl.SelectedItem.Item.Count -= addCount
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
			pl.Inventory.Items[slot].Count += addCount

		} else {
			// Different types: swap slot and held (left-click only; right-click
			if int(heldItem.Count) <= inventory.MaxStack {
				swapped := slotItem
				pl.Inventory.Place(heldItem, slot)
				pl.SelectedItem.SetItem(swapped, slot, 0, rightClick)
			}
		}
	}
	result := crafting.Craft2x2(pl.Inventory.GetCrafting2x2())
	if result.TypeId != -1 {
		sendSetSlot(pl.Connection, 0, 0, inventory.NewItem(result.TypeId, result.Count, result.Metadata))
	}
}

func craftInWorkbench(pl *player.Player, shift, rightClick bool) {
	hasHeld := pl.SelectedItem.Selected
	if hasHeld && !shift {
		// noop if item in hand
		return
	}
	// stateful crafter
	//result := crafting.New3x3CrafterV2().Craft(pl.Workbench.GetGrid())

	// stateless conditional crafting
	result := crafting.Craft3x3(pl.Workbench.GetGrid())
	resultItem := inventory.NewItem(result.TypeId, result.Count, result.Metadata)
	if result.TypeId != -1 {
		// Consume one item from each occupied grid slot.
		for slot := int16(1); slot <= 9; slot++ {
			pl.Workbench.RemoveOneFromGridSlot(slot)
		}

		if shift {
			//pl.SelectedItem.Clear()
			res := pl.Inventory.AddItemHotbarFromRightToLeft(resultItem.TypeId, result.Metadata, resultItem.Count)
			if !res {
				pl.Inventory.AddItem(resultItem.TypeId, result.Metadata, resultItem.Count)
			}
		} else {
			pl.SelectedItem.SetItem(resultItem, 0, 0, rightClick)
		}
	}
}

func craftInInventory(pl *player.Player, shift, rightClick bool) {
	hasHeld := pl.SelectedItem.Selected
	if hasHeld && !shift {
		// noop if item in hand
		return
	}
	inv := pl.Inventory
	result := crafting.Craft2x2(inv.GetCrafting2x2())

	resultItem := inventory.NewItem(result.TypeId, result.Count, result.Metadata)
	if result.TypeId != -1 {
		for slot := int16(1); slot <= 4; slot++ {
			if inv.PeekItem(slot).TypeId != -1 {
				inv.RemoveOneFromSlot(slot)
			}
		}

		if shift {
			//pl.SelectedItem.Clear()
			res := pl.Inventory.AddItemHotbarFromRightToLeft(resultItem.TypeId, result.Metadata, resultItem.Count)
			if !res {
				pl.Inventory.AddItem(resultItem.TypeId, result.Metadata, resultItem.Count)
			}
		} else {
			pl.SelectedItem.SetItem(resultItem, 0, 0, rightClick)
		}
	}
}

func shiftClickWorkbench(pl *player.Player, slot int16) {
	sourceItem := pl.Workbench.GetGridItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}

	if pl.Workbench.IsCraftingSlot(slot) {
		shiftMoveToRegionInWorkbench(pl, slot, inventory.MainInventoryStart, inventory.HotbarEnd)
	} else if pl.Inventory.IsHotbarSlot(slot) {
		shiftMoveToRegionInWorkbench(pl, slot, inventory.MainInventoryStart, inventory.MainInventoryEnd)
	} else {
		shiftMoveToRegionInWorkbench(pl, slot, inventory.HotbarStart, inventory.HotbarEnd)
	}
}

// It seems that in decompiled Beta 1.7.3 server, Shift is equivalent for right / left click
// If Beta Tweaks is used, it sends different packets to simulate Modern Minecraft shift click
func shiftClick(pl *player.Player, slot int16) {
	sourceItem := pl.Inventory.PeekItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}

	if pl.Inventory.IsCraftingSlot(slot) {
		shiftMoveToRegion(pl, slot, inventory.MainInventoryStart, inventory.HotbarEnd)
	} else if pl.Inventory.IsHotbarSlot(slot) {
		shiftMoveToRegion(pl, slot, inventory.MainInventoryStart, inventory.MainInventoryEnd)
	} else {
		shiftMoveToRegion(pl, slot, inventory.HotbarStart, inventory.HotbarEnd)
	}
}

// if you play it, you realize that shift click moves it to the first empty slot in either
// inventory or hotbar
func shiftMoveToRegion(pl *player.Player, sourceSlot int16, regionStart, regionEnd int) {
	// merge onto partial stacks of the same type (skip for non-stackable items).
	if inventory.IsStackable(pl.Inventory.PeekItem(sourceSlot).TypeId) {
		for i := regionStart; i <= regionEnd && pl.Inventory.PeekItem(sourceSlot).TypeId != -1; i++ {
			target := pl.Inventory.PeekItem(int16(i))
			source := pl.Inventory.PeekItem(sourceSlot)
			if target.TypeId == source.TypeId && target.Metadata == source.Metadata && target.Count < inventory.MaxStack {
				room := inventory.MaxStack - int(target.Count)
				move := int(source.Count)
				if move > room {
					move = room
				}
				pl.Inventory.Items[i].Count += byte(move)
				pl.Inventory.Items[sourceSlot].Count -= byte(move)
				if pl.Inventory.Items[sourceSlot].Count == 0 {
					pl.Inventory.Items[sourceSlot] = inventory.NewItem(-1, 0, 0)
				}
			}
		}
	}

	// occurs when shift click created a stack, rest is also moved
	// is also triggered if previous move didn't do anything, so nother item of the same existed
	if pl.Inventory.PeekItem(sourceSlot).TypeId != -1 {
		for i := regionStart; i <= regionEnd; i++ {
			if pl.Inventory.PeekItem(int16(i)).TypeId == -1 {
				pl.Inventory.Move(sourceSlot, int16(i))
				break
			}
		}
	}
	// if inventory full of stacks of same item type, let's not do anything.
}

func shiftMoveToRegionInWorkbench(pl *player.Player, sourceSlot int16, regionStart, regionEnd int) {
	// merge onto partial stacks of the same type (skip for non-stackable items).
	if inventory.IsStackable(pl.Workbench.GetGridItem(sourceSlot).TypeId) {
		for i := regionStart; i <= regionEnd && pl.Workbench.GetGridItem(sourceSlot).TypeId != -1; i++ {
			target := pl.Inventory.PeekItem(int16(i))
			source := pl.Workbench.GetGridItem(sourceSlot)
			if target.TypeId == source.TypeId && target.Metadata == source.Metadata && target.Count < inventory.MaxStack {
				room := inventory.MaxStack - int(target.Count)
				move := int(source.Count)
				if move > room {
					move = room
				}
				pl.Inventory.Items[i].Count += byte(move)
				pl.Workbench.Grid[sourceSlot-1].Count -= byte(move)
				if pl.Workbench.Grid[sourceSlot-1].Count == 0 {
					pl.Workbench.Grid[sourceSlot-1] = inventory.NewItem(-1, 0, 0)
				}
			}
		}
	}

	// occurs when shift click created a stack, rest is also moved
	// is also triggered if previous move didn't do anything, so nother item of the same existed
	if pl.Workbench.GetGridItem(sourceSlot).TypeId != -1 {
		for i := regionStart; i <= regionEnd; i++ {
			if pl.Inventory.PeekItem(int16(i)).TypeId == -1 {
				inventory.MoveFromWorkbenchToInventory(&pl.Workbench, &pl.Inventory, sourceSlot, int16(i))
				break
			}
		}
	}
	// if inventory full of stacks of same item type, let's not do anything.
}

func acceptTransaction(connection net.Conn, p packets.WindowClickInPacket) {
	out := packets.TransactionOutPacket{
		WindowId:     0,
		ActionNumber: p.ActionNumber,
		Accepted:     true,
	}
	connection.Write(out.Serialize())
}

func handleCloseWindowInPacket(p packets.CloseWindowInPacket) {
	log.Printf("CloseWindow: %+v", p)
}
