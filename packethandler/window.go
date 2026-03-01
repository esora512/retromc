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
	if p.Shift {
		sendInventory(pl.Connection, pl)
		pl.SelectedItem.Selected = false
		return
	}

	if pl.SelectedItem.Selected && pl.SelectedItem.ActionNumber == p.ActionNumber-1 {
		log.Printf("Swapping slots %d and %d", pl.SelectedItem.Slot, p.Slot)
		pl.Inventory.Swap(pl.SelectedItem.Slot, p.Slot)
		pl.SelectedItem.Selected = false

	}
	if p.ItemID > 0 {
		log.Printf("Selecting item %d in slot %d", p.ItemID, p.Slot)
		pl.SelectedItem = player.SelectedItem{
			Item: player.Item{
				TypeId: p.ItemID,
				Count:  p.ItemCount,
				Uses:   p.ItemUses,
			},
			Selected:     true,
			ActionNumber: p.ActionNumber,
			Slot:         p.Slot}
	}
}
