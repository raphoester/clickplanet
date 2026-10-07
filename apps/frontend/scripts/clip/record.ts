import {writeFileSync} from "node:fs"
import {Timestamp} from "@bufbuild/protobuf"
import {GetReplayResponse, ReplayedEvent} from "../../src/gen/grpc/planet/v1/admin_pb.ts"
import {GetMapResponse} from "../../src/gen/grpc/planet/v1/planet_pb.ts"
import {newClickServiceClient} from "../../src/backends/planetBackend.ts"
import {NO_TIMEOUT} from "../../src/backends/transport.ts"

const args = process.argv.slice(2)
const flag = (name: string, fallback?: string) => {
    const i = args.indexOf(`--${name}`)
    return i === -1 ? fallback : args[i + 1]
}

const out = flag("out")
if (!out || args.includes("--help")) {
    console.error(`usage: npm run clip:record -- --out <replay.json> [--minutes 10] [--api https://api.clickplanet.lol]

Reads the map and follows the public live stream for a while, and writes what it saw as a replay.
A replay of the last hours comes from the operator tool instead: see GetReplay in deploy/vps/README.md.`)
    process.exit(1)
}

const TILES_PER_BATCH = 10_000
const KEPT = new Set(["tileUpdate", "bombDropped", "tilesSpread", "tilesEnclosed"])

const client = newClickServiceClient({baseUrl: flag("api", "https://api.clickplanet.lol")!, timeoutMs: 30_000})
const minutes = Number(flag("minutes", "10"))

const since = new Date()
const {density: maxIndex} = await client.mapDensity({})

const codes: string[] = [""]
const codeOf = new Map([["", 0]])
const tiles = new Uint8Array(maxIndex * 2)
const shields = new Map<number, number>()
for (let start = 1; start <= maxIndex; start += TILES_PER_BATCH) {
    const batch = await client.getMap({startTileId: start, endTileId: Math.min(start + TILES_PER_BATCH, maxIndex)})
    const view = new DataView(batch.tiles.buffer, batch.tiles.byteOffset, batch.tiles.byteLength)
    for (let offset = 0; offset + 1 < batch.tiles.byteLength; offset += 2) {
        const owner = batch.codes[view.getUint16(offset, true)]
        let code = codeOf.get(owner)
        if (code === undefined) {
            code = codes.length
            codes.push(owner)
            codeOf.set(owner, code)
        }
        const tile = batch.startTileId + offset / 2
        tiles[(tile - 1) * 2] = code & 0xff
        tiles[(tile - 1) * 2 + 1] = code >> 8
    }
    for (const {tileId, shields: count} of batch.shields) shields.set(tileId, count)
}
console.log(`read the map: ${maxIndex} tiles, ${codes.length - 1} flags`)

const events: ReplayedEvent[] = []
const stop = new AbortController()
setTimeout(() => stop.abort(), minutes * 60_000)
try {
    for await (const event of client.listenForEvents({}, {signal: stop.signal, timeoutMs: NO_TIMEOUT})) {
        if (!event.event.case || !KEPT.has(event.event.case)) continue
        events.push(new ReplayedEvent({at: Timestamp.now(), event}))
        if (events.length % 100 === 0) console.log(`${events.length} events`)
    }
} catch (error) {
    if (!stop.signal.aborted) throw error
}

const replay = new GetReplayResponse({
    opening: new GetMapResponse({
        startTileId: 1,
        codes,
        tiles,
        shields: [...shields].map(([tileId, count]) => ({tileId, shields: count})),
    }),
    events,
    since: Timestamp.fromDate(since),
    until: Timestamp.now(),
})
writeFileSync(out, replay.toJsonString())
console.log(`saved ${events.length} events over ${minutes} minutes to ${out}`)
