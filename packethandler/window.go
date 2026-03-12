package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/crafting"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// Refer to method: func_27085_a in Container.java in decompiled Minecraft Beta 1.7.3 server
// Refer to method: func_20007_a in NetServerHandler.java in decompiled Minecraft Beta 1.7.3 server
func handleWindowClickInPacket(connection net.Conn, p packets.WindowClickInPacket, pl *player.Player) {
	if p.WindowId != 0 {
		return
	}

	p.Print()

	rightClick := p.RightClick == 1
	shift := p.Shift
	slot := p.Slot

	if slot == 0 {
		craftInInventory(pl, shift, rightClick)
		acceptTransaction(connection, p)
		return
	}

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
		sameType := slotItem.TypeId == heldItem.TypeId

		if sameType {
			// Merge held into slot (up to MaxStack).
			addCount := heldItem.Count
			if rightClick {
				addCount = 1
			}
			// Cap by slot limit and item max stack.
			room := player.MaxStack - int(slotItem.Count)
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
			if int(heldItem.Count) <= player.MaxStack {
				swapped := slotItem
				pl.Inventory.Place(heldItem, slot)
				pl.SelectedItem.SetItem(swapped, slot, 0, rightClick)
			}
		}
	}
}

func craftInInventory(pl *player.Player, shift, rightClick bool) {
	inv := pl.Inventory
	result := crafting.GetCrafter2x2().Craft(inv.GetCrafting2x2())

	resultItem := player.NewItem(result, 1)
	if result != -1 {
		log.Println("Craft")
		for slot := int16(1); slot <= 4; slot++ {
			if inv.PeekItem(slot).TypeId != -1 {
				inv.RemoveOneFromSlot(slot)
			}
		}

		if shift {
			pl.Inventory.AddItem(resultItem.TypeId)
		} else {
			pl.SelectedItem.SetItem(resultItem, 0, 0, rightClick)
		}
	}
}

// It seems that in decompiled Beta 1.7.3 server, Shift is equivalent for right / left click
// If Beta Tweaks is used, it sends different packets to simulate Modern Minecraft shift click
func shiftClick(pl *player.Player, slot int16) {
	sourceItem := pl.Inventory.PeekItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}

	if pl.Inventory.IsHotbarSlot(slot) {
		shiftMoveToRegion(pl, slot, player.MainInventoryStart, player.MainInventoryEnd)
	} else {
		shiftMoveToRegion(pl, slot, player.HotbarStart, player.HotbarEnd)
	}
}

// if you play it, you realize that shift click moves it to the first empty slot in either
// inventory or hotbar
func shiftMoveToRegion(pl *player.Player, sourceSlot int16, regionStart, regionEnd int) {
	// merge onto partial stacks of the same type.
	for i := regionStart; i <= regionEnd && pl.Inventory.PeekItem(sourceSlot).TypeId != -1; i++ {
		target := pl.Inventory.PeekItem(int16(i))
		source := pl.Inventory.PeekItem(sourceSlot)
		if target.TypeId == source.TypeId && target.Count < player.MaxStack {
			room := player.MaxStack - int(target.Count)
			move := int(source.Count)
			if move > room {
				move = room
			}
			pl.Inventory.Items[i].Count += byte(move)
			pl.Inventory.Items[sourceSlot].Count -= byte(move)
			if pl.Inventory.Items[sourceSlot].Count == 0 {
				pl.Inventory.Items[sourceSlot] = player.NewItem(-1, 0)
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

func acceptTransaction(connection net.Conn, p packets.WindowClickInPacket) {
	out := packets.TransactionOutPacket{
		WindowId:     0,
		ActionNumber: p.ActionNumber,
		Accepted:     true,
	}
	connection.Write(out.Serialize())
}
