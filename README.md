# retromc
A Mincraft Beta 1.7.3 server written in Go I [forked](https://github.com/leNicDev/retromc).

## Plan
* Create a functional Minecraft Beta 1.7.3 version
* Fiddle with world generation to create interesting worlds
* Play and have fun on that server

## Side goals
* Learn more about Minecraft networking

## Emojis
* 🆗: Works mostly, has some bugs with low priority.
* ✔️: Seems to work properly
* ❌: Considered not ready/done
* 🚧: Working on it right now
* 🐞: Temporary tracking of bugs that should be fixed
* 🕸️: Low priority bugs that will be fixed eventually/later
* 🎯: Temporary goals that should be implemented during 🚧 state to get to 🆗/✔️ states
* ℹ️: Information

## Status
* 🆗: Inventory Management (Clicking & Moving)
    * 🕸️: Issue involving Beta Tweaks, too many packets sends and visible in client.
* ✔️: Crafting
* ✔️: Multiplayer
    * ℹ️: Two client POV match what is happening on individual client (hopefully...)
* 🆗: Block Placement & Mining
    * 🕸️: Some delays and undos of block placement
* ❌: Physics
    * 🚧: Minecart
        * 🐞: Lighting issues
        * 🐞: Player is moving inside minecart when entering from further distance
        * 🐞: Collision still not "smooth" enough but not high priority
        * 🎯: Should work with powered rails
        * 🎯: Should be breakable & collectable
    * ❌: Boat
    * ❌: Water/Lava
    * ❌: Sand/Gravel gravity
* ❌: World Generation
    * ℹ️: Right now it's 16x16 stone platform, so a ~~sand~~ *stone* box.
* ❌: Dropping Items (Entity Rendering) 

## References
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