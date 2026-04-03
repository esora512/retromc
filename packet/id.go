package packet

const (
	KeepAlive                 byte = 0x00
	LoginRequest              byte = 0x01
	Handshake                 byte = 0x02
	ChatMessage               byte = 0x03
	TimeUpdate                byte = 0x04
	EntityEquipment           byte = 0x05
	SpawnPosition             byte = 0x06
	PlayerPosition            byte = 0x0b
	PlayerPositionAndLook     byte = 0x0d
	PreChunk                  byte = 0x32
	MapChunk                  byte = 0x33
	SetSlot                   byte = 0x67
	WindowItems               byte = 0x68
	PlayerOnGround            byte = 0x0a
	PlayerDigging             byte = 0x0e
	PlayerBlockPlacement      byte = 0x0f
	PlayerLook                byte = 0x0c
	EntityLook                byte = 0x20
	EntityAction              byte = 0x13
	PlayerAnimation           byte = 0x12
	HoldingChange             byte = 0x10
	BlockChange               byte = 0x35
	WindowClick               byte = 0x66
	SetHealth                 byte = 0x08
	Respawn                   byte = 0x09
	Transaction               byte = 0x6a
	OpenInventory             byte = 0x64
	CloseWindow               byte = 0x65
	EntityPositionAndRotation byte = 0x21
	SpawnPlayerEntity         byte = 0x14
	SetEquipment              byte = 0x05
	InteractWithEntity        byte = 0x07
	EntityPosition            byte = 0x1F
	DespawnEntity             byte = 0x1D
	Disconnect                byte = 0xFF
	EntityMetadata            byte = 0x28
	SpawnObject               byte = 0x17
	AddPassenger              byte = 0x27
	TeleportEntity            byte = 0x22
)
