# retromc
A Mincraft Beta 1.7.3 server written in Go I [forked](https://github.com/leNicDev/retromc).

## Goal
* Create a functional Minecraft Beta 1.7.3 server 
* Support initial Skyblock gameplay in a world with no mobs
* Play and have fun on that server

## Side goals
* Learn more about Minecraft networking
* Fiddle with world generation to explore liminal worlds (distant goal)

## References / Help
* https://pixelbrush.dev/beta-wiki/ (has protocol information; may help in improving this build)
* https://minecraft.wiki/w/Java_Edition_protocol?oldid=2769711 (more accurate protocol information)
* https://github.com/OfficialPixelBrush/BetrockServer (a functional Beta 1.7.3 server written in C++; may be a good reference)
* https://wiki.retromc.org/B1.7.3_data_values (beta 1.7.3 block ids)
* https://github.com/MCPHackers/RetroMCP-Java (decompiling beta 1.7.3 server and playing with code)
* https://github.com/p2r3/bareiron (minimalist C Minecraft server, inspiration / some parts copied)
* https://web.archive.org/web/20110902073903/http://www.minecraftwiki.net/wiki/Crafting (Crafting recipes for Beta 1.7.3)
* https://github.com/jacobo-mc/mc_b1.7.3_release/blob/main/1.7.3-LTS/src/minecraft_server/net/minecraft/src/CraftingManager.java (Crafting recipes, how they were implemented in decompiled code)
* Packet Inspection with `tshark`
    * Run `sudo tshark -i lo -f "host 127.0.0.1 and tcp port 25565"` on a detached screen
    * Run `python3 BetaPacketPlainTextifier.py -v` (tool can be found [here](https://github.com/OfficialPixelBrush/BetaPacketPlainTextifier))
    * Allows you to inspects packets between client & server