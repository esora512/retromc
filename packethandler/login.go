package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleLoginRequestInPacket(connection net.Conn, p packets.LoginRequestInPacket, world *level.World, pl *player.Player) {
	pl.Username = p.Username
	sendLoginResponse(connection, pl)
	sendChunks(connection, world)
	sendInventory(connection, pl)
	sendPlayerPositionAndLook(connection)
	spawnPacket := packets.SpawnPlayerEntityPacket(pl)
	// Inform other players of the new player
	world.MulticastPacket(spawnPacket, pl)
	world.ForEachPlayer(func(other *player.Player) {
		if other == pl {
			return
		}
		packets.SetEquipment(pl, func(b []byte) {
			other.Connection.Write(b)
		})
		chatPacket := packets.ChatMessagePacket{
			Message: "\u00a7e" + pl.Username + " joined the game",
		}
		other.Connection.Write(chatPacket.Serialize())
	})

	// Inform the new player of other players
	world.ForEachPlayer(func(other *player.Player) {
		if other == pl {
			return
		}
		spawnPacket := packets.SpawnPlayerEntityPacket(other)
		pl.Connection.Write(spawnPacket)
		packets.SetEquipment(other, func(b []byte) {
			pl.Connection.Write(b)
		})
	})
}

func sendLoginResponse(connection net.Conn, pl *player.Player) {
	// create login response packet
	outPacket := packets.LoginResponseOutPacket{
		EntityId:  pl.EntityId,
		MapSeed:   0,
		Dimension: 0,
	}
	outData := outPacket.Serialize()

	// write login response packet
	connection.Write(outData)
	pl.LoggedIn = true
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

func sendInventory(connection net.Conn, pl *player.Player) {
	// Apply preset items into the player's in-memory inventory first,
	// then send the full inventory in a single WindowItems packet.
	presetInventory(&pl.Inventory)

	windowItemsPacket := packets.WindowItemsOutPacket{
		WindowId: 0, // 0 = player inventory
		Count:    int16(pl.Inventory.Size),
		Payload:  pl.Inventory,
	}
	connection.Write(windowItemsPacket.Serialize())
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
