# retromc
![Go](https://img.shields.io/badge/Language-Go1.25-5E96CF)
![Issues](https://img.shields.io/github/issues/esora512/retromc)
![Pull requests](https://img.shields.io/github/issues-pr/esora512/retromc)

A Mincraft Beta 1.7.3 server written in Go I [forked](https://github.com/leNicDev/retromc).


## Goal
* Create a functional Minecraft Beta 1.7.3 server 
* Support basic liminal-space like world gen
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
* NBT Reference: https://github.com/OfficialPixelBrush/BetrockPlusPlus/blob/main/src/bpp_shared/world/storage/region.cpp#L364

## Golang
* https://github.com/sasha-s/go-deadlock (Helps to debug deadlocks; became more relevant in this codebase)

## Deployment
Get a vm and run:
```
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz && echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
```
Then just clone the repo and run `bash build.sh`
Finally, run the server with `./retromc --host 0.0.0.0`

## External Vibe-coded world gen
It is possible to get somewhat identical world generation to B1.7.3 by having a server that has this already and just creating a tool that spits out the chunks: I VIBE-CODED this using BetrockPlusPlus: https://github.com/esora512/BetrockPlusPlus/tree/vibe-gen/chunkgen
* I have no idea about C++ but I know their world gen is solid, so I just asked Claude Code to generate a tool that uses their code and spits out the data I need for my chunk generation to work.
* It is not 100% perfect but good enough for now, so I can more or less postpone work on world generation while still having it more or less. Plan is that me in the future (or perhaps other contributor) sets up world gen but at the moment I do not care.

To use this world gen approach, you have to clone my fork of the BetrockPlusPlus repo, checkout into the `vibe-gen` branch, build the binary and move it wherever you want, for example `/bin/chunkgen`
Then you can access it via:
```sh
./retromc --external-chunkgen-bin bin/chunkgen 
```

You can download the binary also from [![Google Drive](https://img.shields.io/badge/Google%20Drive-Open%20File-4285F4?logo=googledrive&logoColor=white)](https://drive.google.com/file/d/1Wrw3ePTMh3mM0Dzk1-4r4pwuzkGMCoQN/view?usp=sharing)

