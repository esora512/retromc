package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/crafting"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// Refer to method: func_27085_a in Container.java in decompiled Minecraft Beta 1.7.3 server
// Refer to method: func_20007_a in NetServerHandler.java in decompiled Minecraft Beta 1.7.3 server
func handleClickSlotPacket(connection net.Conn, p packets.ClickSlotPacket, world *level.World, pl *player.Player, tracker *level.EntityTracker) {
	rightClick := p.RightClick == 1
	shift := p.Shift
	slot := p.Slot
	windowId := p.WindowId

	if slot == -999 {
		if !pl.SelectedItem.Selected {
			acceptTransaction(connection, p)
			return
		}

		typeId := pl.SelectedItem.Item.TypeId
		metadata := pl.SelectedItem.Item.Metadata
		var dropCount byte

		if rightClick {
			dropCount = 1
			pl.SelectedItem.Item.Count--
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
		} else {
			dropCount = pl.SelectedItem.Item.Count
			pl.SelectedItem.Selected = false
		}

		acceptTransaction(connection, p)
		DropItemFromPlayer(world, pl, typeId, metadata, dropCount, tracker)
		return
	}

	if windowId == 1 && pl.InventoryType == player.ChestInventory {
		chest := world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z)
		if slot >= int16(chest.Size) {
			windowId = 0
			slot = chest.ShiftSlot(slot)
		} else {
			if shift {
				shiftClickChest(pl, slot, world)
			} else {
				chestClick(pl, slot, rightClick, world)
			}
			broadcastChestContents(world, pl, chest)
			acceptTransaction(connection, p)
			return
		}
	}

	if windowId == 1 && pl.InventoryType == player.DispenserInventory {
		dispenser := world.GetDispenser(pl.Dispenser.X, pl.Dispenser.Y, pl.Dispenser.Z)
		if dispenser == nil {
			return
		}
		if slot >= int16(dispenser.Size) {
			windowId = 0
		} else {
			if shift {
				// noop
				// CLient with Beta Tweaks does not send shift click for dispenser
			} else {
				dispenserClick(pl, slot, rightClick, world)
			}
			broadcastDispenserContents(world, pl, dispenser)
			acceptTransaction(connection, p)
			return
		}
	}

	if windowId == 1 && pl.InventoryType == player.FurnaceInventory {
		furnace := world.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z)
		if furnace == nil {
			return
		}
		if slot >= int16(furnace.Size) {
			windowId = 0
			slot = furnace.ShiftSlot(slot)
		} else {
			if shift {
				shiftClickFurnace(pl, slot, world)
			} else {
				slotItem := furnace.PeekItem(slot)
				hasHeld := pl.SelectedItem.Selected
				slotEmpty := slotItem.TypeId == -1
				if slot == 2 {
					// log.Println("Slot 2 click furnace")
					// log.Printf("Has Held %t, SlotEmpty %t", hasHeld, slotEmpty)
					if hasHeld || slotEmpty {
						// noop if item in hand
						return
					}
					furnaceOutputClick(pl, slot, rightClick, world)
				} else {
					// only bottom and top slot are clickable
					//log.Println("Regular furnace click")
					furnaceClick(pl, slot, rightClick, world)
				}
			}
			broadcastFurnaceContents(world, pl, furnace)
			acceptTransaction(connection, p)
			return
		}
	}

	// Workbench actions
	if windowId == 1 && pl.InventoryType == player.WorkbenchInventory {
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
			//pl.Workbench.Print()
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
			shiftClick(pl, slot, world)
			// shiftClick may move items from the player inventory into an open chest
			if pl.InventoryType == player.ChestInventory {
				chest := world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z)
				if chest != nil {
					broadcastChestContents(world, pl, chest)
				}
			}
		} else {
			normalClick(pl, slot, rightClick, world)
		}

		acceptTransaction(connection, p)
		//pl.SelectedItem.Print()
		//pl.Inventory.Print()
	}
}

func craftInWorkbench(pl *player.Player, shift, rightClick bool) {
	result := crafting.Craft3x3(pl.Workbench.GetGrid())

	if result.TypeId == -1 {
		return
	}

	held := pl.SelectedItem
	hasHeld := held.Selected

	if hasHeld && !shift {
		heldItem := held.Item
		sameItem := heldItem.TypeId == result.TypeId && heldItem.Metadata == result.Metadata

		var maxStack int
		if inventory.IsStackable(result.TypeId) {
			maxStack = 64
		} else {
			maxStack = 1
		}
		hasRoom := sameItem && heldItem.Count+result.Count <= byte(maxStack)

		if !sameItem || !hasRoom {
			return
		}
	}

	resultItem := inventory.NewItem(result.TypeId, result.Count, result.Metadata)

	for slot := int16(1); slot <= 9; slot++ {
		pl.Workbench.RemoveOne(slot)
	}

	if shift {
		pl.Inventory.AddItem(resultItem.TypeId, result.Metadata, resultItem.Count)
	} else {
		if hasHeld {
			merged := inventory.NewItem(result.TypeId, held.Item.Count+result.Count, result.Metadata)
			pl.SelectedItem.SetItem(merged, 0, 0, rightClick)
		} else {
			pl.SelectedItem.SetItem(resultItem, 0, 0, rightClick)
		}
	}
}

func craftInInventory(pl *player.Player, shift, rightClick bool) {
	inv := pl.Inventory
	result := crafting.Craft2x2(inv.GetCrafting2x2())

	if result.TypeId == -1 {
		return
	}

	held := pl.SelectedItem
	hasHeld := held.Selected

	if hasHeld && !shift {
		heldItem := held.Item
		sameItem := heldItem.TypeId == result.TypeId && heldItem.Metadata == result.Metadata

		var maxStack int
		if inventory.IsStackable(result.TypeId) {
			maxStack = 64
		} else {
			maxStack = 1
		}
		hasRoom := sameItem && heldItem.Count+result.Count <= byte(maxStack)

		if !sameItem || !hasRoom {
			return
		}
	}

	resultItem := inventory.NewItem(result.TypeId, result.Count, result.Metadata)

	for slot := int16(1); slot <= 4; slot++ {
		if inv.PeekItem(slot).TypeId != -1 {
			inv.RemoveOne(slot)
		}
	}

	if shift {
		pl.Inventory.AddItem(result.TypeId, result.Metadata, result.Count)
	} else {
		if hasHeld {
			merged := inventory.NewItem(result.TypeId, held.Item.Count+result.Count, result.Metadata)
			pl.SelectedItem.SetItem(merged, 0, 0, rightClick)
		} else {
			pl.SelectedItem.SetItem(resultItem, 0, 0, rightClick)
		}
	}
}

func shiftClickFurnace(pl *player.Player, slot int16, world *level.World) {
	furnace := world.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z)
	if furnace == nil {
		return
	}
	sourceItem := furnace.PeekItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}
	var sourceContainer inventory.ItemContainer = furnace
	var targetContainer inventory.ItemContainer = &pl.Inventory
	shiftMoveToRegion(slot, inventory.MainInventoryStart, inventory.HotbarEnd, sourceContainer, targetContainer)
	//furnace.Print()
}

func shiftClickChest(pl *player.Player, slot int16, world *level.World) {
	chest := world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z)
	if chest == nil {
		return
	}
	sourceItem := chest.PeekItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}

	var sourceContainer inventory.ItemContainer = chest
	var targetContainer inventory.ItemContainer = &pl.Inventory
	check := inventory.MoveFromSourceToTargetContainer(sourceContainer, targetContainer, slot, inventory.HotbarEnd, inventory.HotbarStart)
	if !check {
		inventory.MoveFromSourceToTargetContainer(sourceContainer, targetContainer, slot, inventory.MainInventoryStart, inventory.MainInventoryEnd)
	}
	//chest.Print()
}

func shiftClickWorkbench(pl *player.Player, slot int16) {
	sourceItem := pl.Workbench.PeekItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}

	var sourceContainer inventory.ItemContainer = &pl.Workbench
	var targetContainer inventory.ItemContainer = &pl.Inventory

	if pl.Workbench.IsCraftingSlot(slot) {
		shiftMoveToRegion(slot, inventory.MainInventoryStart, inventory.HotbarEnd, sourceContainer, targetContainer)
	} else if pl.Inventory.IsHotbarSlot(slot) {
		shiftMoveToRegion(slot, inventory.MainInventoryStart, inventory.MainInventoryEnd, sourceContainer, targetContainer)
	} else {
		shiftMoveToRegion(slot, inventory.HotbarStart, inventory.HotbarEnd, sourceContainer, targetContainer)
	}
}

// It seems that in decompiled Beta 1.7.3 server, Shift is equivalent for right / left click
// If Beta Tweaks is used, it sends different packets to simulate Modern Minecraft shift click
func shiftClick(pl *player.Player, slot int16, world *level.World) {
	sourceItem := pl.Inventory.PeekItem(slot)
	if sourceItem.TypeId == -1 {
		return
	}

	var sourceContainer inventory.ItemContainer = &pl.Inventory
	var targetContainer inventory.ItemContainer = &pl.Inventory

	if pl.Inventory.IsCraftingSlot(slot) {
		shiftMoveToRegion(slot, inventory.MainInventoryStart, inventory.HotbarEnd, sourceContainer, targetContainer)
	} else if pl.InventoryType == player.ChestInventory {
		chest := world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z)
		targetContainer = chest
		chestEnd := chest.Size - 1
		shiftMoveToRegion(slot, inventory.ChestStart, int(chestEnd), sourceContainer, targetContainer)

	} else if pl.Inventory.IsHotbarSlot(slot) {
		shiftMoveToRegion(slot, inventory.MainInventoryStart, inventory.MainInventoryEnd, sourceContainer, targetContainer)
	} else {
		shiftMoveToRegion(slot, inventory.HotbarStart, inventory.HotbarEnd, sourceContainer, targetContainer)
	}
}

// if you play it, you realize that shift click moves it to the first empty slot in either
// inventory or hotbar
func shiftMoveToRegion(sourceSlot int16, regionStart, regionEnd int, sourceContainer, targetContainer inventory.ItemContainer) {
	// merge onto partial stacks of the same type (skip for non-stackable items).
	if inventory.IsStackable(sourceContainer.PeekItem(sourceSlot).TypeId) {
		for i := regionStart; i <= regionEnd && sourceContainer.PeekItem(sourceSlot).TypeId != -1; i++ {
			target := targetContainer.PeekItem(int16(i))
			source := sourceContainer.PeekItem(sourceSlot)
			maxStack := inventory.MaxStack(target.TypeId)
			if target.TypeId == source.TypeId && target.Metadata == source.Metadata && target.Count < byte(maxStack) {
				room := maxStack - int(target.Count)
				move := int(source.Count)
				if move > room {
					move = room
				}
				targetContainer.AddCount(int16(i), byte(move))
				sourceContainer.AddCount(sourceSlot, -byte(move))

				if sourceContainer.PeekItem(sourceSlot).Count == 0 {
					sourceContainer.SetEmpty(sourceSlot)
				}
			}
		}
	}
	// occurs when shift click created a stack, rest is also moved
	// is also triggered if previous move didn't do anything, so nother item of the same existed
	inventory.MoveFromSourceToTargetContainer(sourceContainer, targetContainer, sourceSlot, regionStart, regionEnd)
	// if inventory full of stacks of same item type, let's not do anything.
}

func furnaceOutputClick(pl *player.Player, slot int16, rightClick bool, world *level.World) {
	furnace := world.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z)
	if furnace == nil {
		return
	}
	slotItem := furnace.PeekItem(slot)
	if slotItem.TypeId == -1 {
		return // nothing to pick up
	}
	if pl.SelectedItem.Selected {
		return // something held — can't place into output slot
	}

	if rightClick {
		pickCount := (slotItem.Count + 1) / 2
		remainCount := slotItem.Count - pickCount
		picked := inventory.NewItem(slotItem.TypeId, pickCount, slotItem.Metadata)
		if remainCount <= 0 {
			furnace.SetEmpty(slot)
		} else {
			furnace.SetItem(slot, slotItem.TypeId, remainCount, slotItem.Metadata)
		}
		pl.SelectedItem.SetItem(picked, slot, 0, true)
	} else {
		picked := inventory.NewItem(slotItem.TypeId, slotItem.Count, slotItem.Metadata)
		furnace.SetEmpty(slot)
		pl.SelectedItem.SetItem(picked, slot, 0, false)
	}
}

func sendSetEquipment(world *level.World, slot, itemId int16, playerId int32) {
	armorSlotMap := map[int16]int16{
		5: 4,
		6: 3,
		7: 2,
		8: 1,
	}
	p := packets.SetEquipmentPacket{
		EntityId:      playerId,
		InventorySlot: armorSlotMap[slot],
		ItemId:        itemId,
		ItemMetadata:  0,
	}
	world.BroadcastPacket(p.Serialize())
}

// Covers all non-shift left- and right-clicks
func guiClick(pl *player.Player, container inventory.ItemContainer, slot int16, rightClick bool) {
	slotItem := container.PeekItem(slot)
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
			container.SetItem(slot, heldItem.TypeId, 1, heldItem.Metadata)
			pl.SelectedItem.Item.Count--
			if pl.SelectedItem.Item.Count == 0 {
				pl.SelectedItem.Selected = false
			}
		} else {
			container.SetItem(slot, heldItem.TypeId, heldItem.Count, heldItem.Metadata)
			pl.SelectedItem.Selected = false
		}

		// something in slot and nothing held
	case !slotEmpty && !hasHeld:
		if rightClick {
			// Decompiled code: var10 = mouseClick == 0 ? var13.stackSize : (var13.stackSize + 1) / 2;
			// For right click, we get (n + 1 / 2)
			pickCount := (slotItem.Count + 1) / 2
			remainCount := slotItem.Count - pickCount
			picked := inventory.NewItem(slotItem.TypeId, pickCount, slotItem.Metadata)
			if remainCount <= 0 {
				container.SetEmpty(slot)
			} else {
				container.SetItem(slot, slotItem.TypeId, remainCount, slotItem.Metadata)
			}
			pl.SelectedItem.SetItem(picked, slot, 0, true)
		} else {
			picked := inventory.NewItem(slotItem.TypeId, slotItem.Count, slotItem.Metadata)
			container.SetEmpty(slot)
			pl.SelectedItem.SetItem(picked, slot, 0, false)
		}

		// something in slot and something held
	case !slotEmpty && hasHeld:
		sameType := slotItem.TypeId == heldItem.TypeId && slotItem.Metadata == heldItem.Metadata
		if sameType && inventory.IsStackable(heldItem.TypeId) {
			addCount := heldItem.Count
			if rightClick {
				addCount = 1
			}
			maxStack := inventory.MaxStack(heldItem.TypeId)
			room := maxStack - int(slotItem.Count)
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
			container.AddCount(slot, addCount)
		} else {
			maxStack := inventory.MaxStack(heldItem.TypeId)
			if int(heldItem.Count) <= maxStack {
				swapped := inventory.NewItem(slotItem.TypeId, slotItem.Count, slotItem.Metadata)
				container.SetItem(slot, heldItem.TypeId, heldItem.Count, heldItem.Metadata)
				pl.SelectedItem.SetItem(swapped, slot, 0, rightClick)
			}
		}
	}
}

func workbenchGridClick(pl *player.Player, slot int16, rightClick bool) {
	guiClick(pl, &pl.Workbench, slot, rightClick)
	result := crafting.Craft3x3(pl.Workbench.GetGrid())
	if result.TypeId != -1 {
		SendSetSlot(pl.Connection, 1, 0, inventory.NewItem(result.TypeId, result.Count, result.Metadata))
	}
}

func normalClick(pl *player.Player, slot int16, rightClick bool, world *level.World) {
	guiClick(pl, &pl.Inventory, slot, rightClick)
	if slot >= 5 && slot <= 8 {
		item := pl.Inventory.PeekItem(slot)
		sendSetEquipment(world, slot, item.TypeId, pl.GetEntityId())
	}
	result := crafting.Craft2x2(pl.Inventory.GetCrafting2x2())
	if result.TypeId != -1 {
		SendSetSlot(pl.Connection, 0, 0, inventory.NewItem(result.TypeId, result.Count, result.Metadata))
	}
}

func chestClick(pl *player.Player, slot int16, rightClick bool, world *level.World) {
	chest := world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z)
	if chest == nil {
		return
	}
	guiClick(pl, chest, slot, rightClick)
	//chest.Print()
}

func dispenserClick(pl *player.Player, slot int16, rightClick bool, world *level.World) {
	dispenser := world.GetDispenser(pl.Dispenser.X, pl.Dispenser.Y, pl.Dispenser.Z)
	if dispenser == nil {
		return
	}
	guiClick(pl, dispenser, slot, rightClick)
	//dispenser.Print()
}

func furnaceClick(pl *player.Player, slot int16, rightClick bool, world *level.World) {
	furnace := world.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z)
	if furnace == nil {
		return
	}
	guiClick(pl, furnace, slot, rightClick)
	//furnace.Print()
}

func acceptTransaction(connection net.Conn, p packets.ClickSlotPacket) {
	out := packets.ContainerTransactionPacket{
		WindowId:     0,
		ActionNumber: p.ActionNumber,
		Accepted:     true,
	}
	connection.Write(out.Serialize())
}

func handleCloseContainerPacket(p packets.CloseContainerPacket, pl *player.Player) {
	pl.InventoryType = player.PlayerInventory
}

func handleWorkbench(p packets.ClickSlotPacket, pl *player.Player, shift, rightClick bool) {
	targetSlot := p.Slot
	if targetSlot == 0 {
		craftInWorkbench(pl, shift, rightClick)
	} else {
		workbenchGridClick(pl, targetSlot, rightClick)
	}
}
