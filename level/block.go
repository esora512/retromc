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

func NewCraftingTable() Block {
	return Block{
		TypeId:   0x3a,
		Metadata: 0x00,
		Light:    0x00,
		SkyLight: 0x00,
	}
}

func NewBlockById(id int16) Block {
	switch id {
	case 0x00:
		return NewAirBlock()
	case 0x01:
		return NewStoneBlock()
	case 0x03:
		return NewDirtBlock()
	case 0x3a:
		return NewCraftingTable()
	default:
		return NewAirBlock()
	}
}
