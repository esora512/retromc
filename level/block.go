package level

type Block struct {
	TypeId   byte
	Metadata byte
	Light    byte
	SkyLight byte
}

func NewAirBlock() Block {
	return Block{
		TypeId:   0x00,
		Metadata: 0x00,
		Light:    0x00,
		SkyLight: 0x0f,
	}
}

func NewStoneBlock() Block {
	return Block{
		TypeId:   0x01,
		Metadata: 0x00,
		Light:    0x00,
		SkyLight: 0x00,
	}
}

func NewDirtBlock() Block {
	return Block{
		TypeId:   0x03,
		Metadata: 0x00,
		Light:    0x00,
		SkyLight: 0x00,
	}
}
