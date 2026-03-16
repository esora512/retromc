# retromc
A Mincraft Beta 1.7.3 server written in Golang I found and forked GitHub: https://github.com/leNicDev/retromc

## Plan
* Create a functional Minecraft Beta 1.7.3 version
* Fiddle with world generation to create interesting worlds
* Play and have fun on that server

## Side goals
* Learn more about Minecraft networking

## Tasks / Goals
* Inventory Management ✔️ (Clicking & Moving)
* Crafting 🚧
* Block Placement & Mining 🚧
    * Block duplication fixed
    * Still a bit buggy; delays and undos of block placement
* Physics ❌
    * Water/Lava
    * Minecart
    * Sand/Gravel gravity
* World Generation ❌
* Dropping Items (Entity Rendering) ❌
* Multiplayer ❌

## References
* https://pixelbrush.dev/beta-wiki/ (has protocol information; may help in improving this build)
* https://minecraft.wiki/w/Java_Edition_protocol?oldid=2769711 (more accurate protocol information)
* https://github.com/OfficialPixelBrush/BetrockServer (a functional Beta 1.7.3 server written in C++; may be a good reference)
* https://wiki.retromc.org/B1.7.3_data_values (beta 1.7.3 block ids)
* https://github.com/MCPHackers/RetroMCP-Java (decompiling beta 1.7.3 server and playing with code)
* https://github.com/p2r3/bareiron (minimalist C Minecraft server, inspiration / some parts copied)