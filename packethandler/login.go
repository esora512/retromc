package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleLoginRequestInPacket(connection net.Conn, p packets.LoginRequestInPacket, world *level.World) {
	log.Printf("Login Request: %+v", p)

	sendLoginResponse(connection)
	sendChunks(connection, world)
	//sendSpawnPosition(connection)
	sendInventory(connection)
	sendPlayerPositionAndLook(connection)
}

func sendLoginResponse(connection net.Conn) {
	// create login response packet
	outPacket := packets.LoginResponseOutPacket{
		EntityId:  0,
		MapSeed:   0,
		Dimension: 0,
	}
	outData := outPacket.Serialize()

	// write login response packet
	connection.Write(outData)
}

// sendChunks sends a 2x2 grid of chunks around the spawn point.
// Each chunk needs a PreChunk (init) followed by its MapChunk (data).
// Chunks are fetched from the world so any already-mutated state is preserved.
func sendChunks(connection net.Conn, world *level.World) {
	for cx := int32(-1); cx <= 0; cx++ {
		for cz := int32(-1); cz <= 0; cz++ {
			// pre-chunk: uses chunk coordinates
			preChunkPacket := packets.PreChunkOutPacket{
				X:    cx,
				Z:    cz,
				Mode: true,
			}
			connection.Write(preChunkPacket.Serialize())

			// map-chunk: X/Z are block coordinates of the chunk's origin
			chunk := world.GetOrCreateChunk(cx, cz)

			mapChunkPacket := packets.MapChunkOutPacket{}
			mapChunkPacket.Apply(*chunk)
			connection.Write(mapChunkPacket.Serialize())
		}
	}
}

func sendSpawnPosition(connection net.Conn) {
	// create spawn position packet
	spawnPositionPacket := packets.SpawnPositionOutPacket{
		X: 0,
		Y: 64,
		Z: 0,
	}
	outData := spawnPositionPacket.Serialize()

	// write spawn position packet
	connection.Write(outData)
}

func sendInventory(connection net.Conn) {
	// create empty player inventory
	inv := player.NewInventory(player.PLAYER_INVENTORY_SIZE)

	// create new window items out packet
	windowItemsPacket := packets.WindowItemsOutPacket{
		WindowId: 0, // 0 = player inventory
		Count:    int16(inv.Size),
		Payload:  inv,
	}
	outData := windowItemsPacket.Serialize()

	// write window items out packet
	connection.Write(outData)

	// put a stone block in the first hotbar slot (slot 36 = leftmost hotbar cell)
	setSlotPacket := packets.SetSlotOutPacket{
		WindowId: 0,                      // 0 = player inventory
		Slot:     36,                     // slots 36-44 are the hotbar; 36 is the first
		Item:     player.NewItem(257, 1), // item 1 = stone, count 64
	}
	outData = setSlotPacket.Serialize()

	// write set slot out packet
	connection.Write(outData)

	setSlotPacket = packets.SetSlotOutPacket{
		WindowId: 0,
		Slot:     37,
		Item:     player.NewItem(1, 64),
	}
	outData = setSlotPacket.Serialize()
	connection.Write(outData)

	setSlotPacket = packets.SetSlotOutPacket{
		WindowId: 0,
		Slot:     38,
		Item:     player.NewItem(326, 1),
	}
	outData = setSlotPacket.Serialize()
	connection.Write(outData)

	setSlotPacket = packets.SetSlotOutPacket{
		WindowId: 0,
		Slot:     39,
		Item:     player.NewItem(327, 1),
	}
	outData = setSlotPacket.Serialize()
	connection.Write(outData)
}

func sendPlayerPositionAndLook(connection net.Conn) {
	const spawnY = 64.0
	// create new player position and look out packet
	packet := packets.PlayerPositionAndLookOutPacket{
		X:        0,
		Y:        spawnY,
		Stance:   spawnY + 2, // Stance MUST be Y + eye height; if Stance < Y client looks up
		Z:        0,
		Yaw:      0,
		Pitch:    0,
		OnGround: true,
	}
	outData := packet.Serialize()

	// write player position and look out packet
	connection.Write(outData)
}
