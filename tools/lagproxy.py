#!/usr/bin/env python3
"""Delaying TCP proxy: lagproxy.py [LISTEN_PORT [TARGET_PORT [ONE_WAY_DELAY_MS [JITTER_MS]]]]
Defaults to 25570 -> 25565, 120ms each way, 0-30ms jitter.
Round-trip latency is about 2x the one-way delay. Byte order is preserved."""
import asyncio, random, socket, sys, time

args = [int(a) for a in sys.argv[1:]] + [25570, 25565, 120, 30][len(sys.argv) - 1:]
listen_port, target_port = args[0], args[1]
delay = args[2] / 1000
jitter = args[3] / 1000


async def pump(reader, writer):
    q = asyncio.Queue()

    async def read():
        last = 0.0
        try:
            while data := await reader.read(65536):
                due = max(time.monotonic() + delay + random.uniform(0, jitter), last)
                last = due  # never reorder
                await q.put((due, data))
        finally:
            await q.put(None)

    async def write():
        try:
            while item := await q.get():
                due, data = item
                if (wait := due - time.monotonic()) > 0:
                    await asyncio.sleep(wait)
                writer.write(data)
                await writer.drain()
        finally:
            writer.close()

    await asyncio.gather(read(), write(), return_exceptions=True)


async def handle(client_r, client_w):
    for w in (client_w,):
        w.get_extra_info("socket").setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
    server_r, server_w = await asyncio.open_connection("127.0.0.1", target_port)
    server_w.get_extra_info("socket").setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
    await asyncio.gather(pump(client_r, server_w), pump(server_r, client_w))


async def main():
    srv = await asyncio.start_server(handle, "0.0.0.0", listen_port)
    print(f"proxy :{listen_port} -> :{target_port}, {delay*1000:.0f}ms each way (+0-{jitter*1000:.0f}ms jitter)")
    async with srv:
        await srv.serve_forever()


asyncio.run(main())
