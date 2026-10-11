#!/usr/bin/env python3
"""
Local TCP <-> Render WebSocket bridge for retromc.

Usage:
    pip install websockets
    python3 bridge.py --remote wss://retromc.onrender.com/ws

Then point your Minecraft client at 127.0.0.1:25565 (or whatever --local is).
"""

import argparse
import asyncio
import logging

import websockets

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(message)s")
log = logging.getLogger("bridge")


async def pipe_tcp_to_ws(reader: asyncio.StreamReader, ws) -> None:
    try:
        while True:
            data = await reader.read(4096)
            if not data:
                break
            await ws.send(data)
    finally:
        await ws.close()


async def pipe_ws_to_tcp(ws, writer: asyncio.StreamWriter) -> None:
    try:
        async for message in ws:
            writer.write(message)
            await writer.drain()
    finally:
        writer.close()


async def handle_client(reader, writer, remote_url: str) -> None:
    peer = writer.get_extra_info("peername")
    log.info("Client connected: %s", peer)
    try:
        async with websockets.connect(remote_url, max_size=None) as ws:
            await asyncio.gather(
                pipe_tcp_to_ws(reader, ws),
                pipe_ws_to_tcp(ws, writer),
            )
    except Exception as exc:
        log.error("Bridge error for %s: %s", peer, exc)
    finally:
        writer.close()
        log.info("Client disconnected: %s", peer)


async def main() -> None:
    parser = argparse.ArgumentParser(description="TCP <-> WebSocket bridge for retromc")
    parser.add_argument("--local", default="127.0.0.1:25565", help="local host:port for the MC client")
    parser.add_argument("--remote", required=True, help="Render WebSocket URL, e.g. wss://retromc.onrender.com/ws")
    args = parser.parse_args()

    local_host, local_port = args.local.split(":")
    server = await asyncio.start_server(
        lambda r, w: handle_client(r, w, args.remote),
        local_host,
        int(local_port),
    )
    log.info("Point your Minecraft client at %s", args.local)
    async with server:
        await server.serve_forever()


if __name__ == "__main__":
    asyncio.run(main())
