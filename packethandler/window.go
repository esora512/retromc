package packethandler

import (
	"log"

"github.com/leNicDev/retromc/packet/packets"
"github.com/leNicDev/retromc/player"
)

func handleWindowClickInPacket(p packets.WindowClickInPacket, pl *player.Player) {
	log.Printf("Window click: %+v", p)
}
