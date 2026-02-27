package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleLoginRequestInPacket(connection net.Conn, p packets.LoginRequestInPacket) {
	log.Printf("Login Request: %+v", p)

	sendLoginResponse(connection)
	sendChunks(connection)
	sendSpawnPosition(connection)
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

// sendChunks sends a 3x3 grid of chunks around the spawn point.
// Each chunk needs a PreChunk (init) followed by its MapChunk (data).
func sendChunks(connection net.Conn) {
	for cx := int32(-1); cx <= 1; cx++ {
		for cz := int32(-1); cz <= 1; cz++ {
			// pre-chunk: uses chunk coordinates
			preChunkPacket := packets.PreChunkOutPacket{
				X:    cx,
				Z:    cz,
				Mode: true,
			}
			connection.Write(preChunkPacket.Serialize())

			// map-chunk: X/Z are block coordinates of the chunk's origin
			chunk := level.NewChunk()
			chunk.X = cx * level.CHUNK_SIZE_X
			chunk.Z = cz * level.CHUNK_SIZE_Z

			mapChunkPacket := packets.MapChunkOutPacket{}
			mapChunkPacket.Apply(chunk)
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

	// create new set slot out packet
	setSlotPacket := packets.SetSlotOutPacket{
		WindowId: 0x81,
		Slot:     -1,
		Item:     player.NewItem(-1, 1),
	}
	outData = setSlotPacket.Serialize()

	// write set slot out packet
	connection.Write(outData)
}

func sendPlayerPositionAndLook(connection net.Conn) {
	// create new player position and look out packet
	packet := packets.PlayerPositionAndLookOutPacket{
		X:        0,
		Y:        80.0,  // feet on top of stone (Y=63)
		Stance:   65.62, // eyes = Y + 1.62 (player eye height)
		Z:        0,
		Yaw:      0,
		Pitch:    0,
		OnGround: true,
	}
	outData := packet.Serialize()

	// write player position and look out packet
	connection.Write(outData)
}
