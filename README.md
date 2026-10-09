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

### Render
When deployed on Render (detected via the `RENDER` env var that Render sets automatically), the server runs as a web service on `$PORT` and tunnels game traffic over a WebSocket at `/ws`. If `KEY_ID`, `APP_KEY` and `B2_BUCKET` are set, the world is restored from Backblaze B2 on startup and backed up every 5 minutes, on shutdown, and on `/save`. We then use a bridge to let the client connect to it. Run the bridge via:
```sh
python3 bridge.py --remote wss://retromc.onrender.com/ws
```

### Discord
Set `DISCORD_TOKEN` (bot token) and `DISCORD_CHANNEL_ID` (chat channel) to bridge in-game chat with a Discord channel; joins, leaves and deaths are posted too. Optionally set `DISCORD_LOG_CHANNEL_ID` to stream server logs to a second channel. Works the same on Render (dashboard env vars) and on a VM (`DISCORD_TOKEN=... DISCORD_CHANNEL_ID=... ./retromc --host 0.0.0.0`).

Bot setup: enable the **Message Content** intent in the Developer Portal and invite the bot with *View Channel*, *Send Messages*, *Embed Links* and *Manage Webhooks* (used to post chat under each player's name and skin face).
