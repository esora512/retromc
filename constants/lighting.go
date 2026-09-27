package constants


var LightEmission [256]byte
var LightOpacity [256]byte

func init() {
	for i := range LightOpacity {
		LightOpacity[i] = 15
	}

	nonOpaque := []int16{
		Air.Value, 6, Glass.Value, 26, PoweredRail.Value, DetectorRail.Value, 29, 30,
		Tallgrass.Value, Deadbush.Value, 33, 34, 36, Dandelion.Value, Rose.Value,
		BrownMushroom.Value, RedMushroom.Value, 44, Torch.Value, Fire.Value, 52,
		WoodenStairs.Value, 55, 59, 63, 64, 65, Rail.Value, CobblestoneStairs.Value,
		68, Lever.Value, 70, 71, 72, RedstoneTorchOff.Value, RedstoneTorchOn.Value,
		StoneButton.Value, SnowLayer.Value, 81, Sugarcane.Value, 85, NetherPortal.Value,
		92, 93, 94, Trapdoor.Value,
	}
	for _, id := range nonOpaque {
		LightOpacity[id] = 0
	}
	LightOpacity[Leaves.Value] = 1
	LightOpacity[WaterFlowing.Value] = 3
	LightOpacity[WaterStill.Value] = 3
	LightOpacity[Ice.Value] = 3 
	LightEmission[LavaFlowing.Value] = 15
	LightEmission[LavaStill.Value] = 15
	LightEmission[Fire.Value] = 15
	LightEmission[Glowstone.Value] = 15
	LightEmission[PumpkinLit.Value] = 15
	LightEmission[LockedChest.Value] = 15 
	LightEmission[Torch.Value] = 14
	LightEmission[FurnaceLit.Value] = 13
	LightEmission[NetherPortal.Value] = 11
	LightEmission[RedstoneOreOn.Value] = 9
	LightEmission[RedstoneRepeaterOn.Value] = 9 
	LightEmission[RedstoneTorchOn.Value] = 7
	LightEmission[BrownMushroom.Value] = 1
}
