import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {GetReplayResponse, ReplayedEvent} from "../gen/grpc/planet/v1/admin_pb.ts"
import {BombDropped, GetMapResponse, PlanetEvent, TileShields, TileUpdate, TilesEnclosed, TilesSpread} from "../gen/grpc/planet/v1/planet_pb.ts"
import {BombDrop, Counts, MapVersion, Ownerships, SpreadClick, Update} from "./backend.ts"
import {ReplayBackend} from "./replayBackend.ts"

const SINCE = Date.UTC(2026, 9, 6, 12)

function at(seconds: number): Timestamp {
    return Timestamp.fromDate(new Date(SINCE + seconds * 1000))
}

function update(seconds: number, tile: number, from: string, to: string): ReplayedEvent {
    return new ReplayedEvent({
        at: at(seconds),
        event: new PlanetEvent({event: {case: "tileUpdate", value: new TileUpdate({
            tileId: tile, previousCountryId: from, countryId: to, clicked: true,
        })}}),
    })
}

function bomb(seconds: number, cleared: number[], struck: {tileId: number, shields: number}[] = []): ReplayedEvent {
    return new ReplayedEvent({
        at: at(seconds),
        event: new PlanetEvent({event: {case: "bombDropped", value: new BombDropped({
            tileId: cleared[0], countryId: "fr", radius: 0.03, clearedTileIds: cleared,
            struck: struck.map((tile) => new TileShields(tile)),
        })}}),
    })
}

function spread(seconds: number): ReplayedEvent {
    return new ReplayedEvent({
        at: at(seconds),
        event: new PlanetEvent({event: {case: "tilesSpread", value: new TilesSpread({
            countryId: "fr", tileId: 1, spreadTileIds: [2],
        })}}),
    })
}

// Tiles 1 and 2 are landmass 1, tile 3 landmass 2; the map opens with 1 and 2 held by pl, 2 with 3 shields.
const LANDMASSES = {assignment: new Uint16Array([1, 1, 2]), count: 3}

function replay(...events: ReplayedEvent[]): ReplayBackend {
    return ReplayBackend.of(new GetReplayResponse({
        opening: new GetMapResponse({
            startTileId: 1, codes: ["", "pl"], tiles: new Uint8Array([1, 0, 1, 0, 0, 0]),
            shields: [new TileShields({tileId: 2, shields: 3})],
        }),
        events,
        since: at(0),
        until: at(60),
    }), LANDMASSES)
}

const ALL = [{start: 1, end: 3}]

async function readOf(backend: ReplayBackend): Promise<Ownerships[]> {
    const read: Ownerships[] = []
    await backend.getOwnerships(ALL, (ownerships) => read.push(ownerships))
    return read
}

const rows = ({countries, landmasses}: Counts) => ({
    countries: Object.fromEntries(countries),
    landmasses: Object.fromEntries([...landmasses].map(([landmass, row]) => [landmass, Object.fromEntries(row)])),
})

describe("a replay", () => {
    it("opens on its map and says its window", async () => {
        const backend = replay()

        const [read] = await readOf(backend)

        expect(read.bindings).toEqual(new Map([[1, "pl"], [2, "pl"]]))
        expect(read.shields).toEqual(new Map([[2, 3]]))
        expect(read.range).toEqual({start: 1, end: 3})
        expect(backend.since).toBe(SINCE)
        expect(backend.until).toBe(SINCE + 60_000)
    })

    it("numbers each step it plays from its opening, as the stream would", async () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [2]), update(3, 3, "", "de"))
        const seen: MapVersion[] = []
        backend.listenForOpenings((version) => seen.push(version))
        backend.listenForUpdatesBatch((updates: Update[]) => seen.push(...updates.map((u) => u.version)))
        backend.listenForBombs((drop: BombDrop) => seen.push(drop.version))

        backend.advanceTo(SINCE + 3_000)

        expect(seen.map(({seq}) => seq)).toEqual([0, 1, 2, 3])
        expect((await readOf(backend))[0].version.seq).toBe(3)
    })

    it("counts what each flag holds, and tells the rows each advance moved", async () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [2]), update(3, 3, "", "de"))
        const changes: Counts[] = []
        backend.listenForHoldings((change) => changes.push(change))

        expect(rows(await backend.getHoldings())).toEqual({countries: {pl: 2}, landmasses: {1: {pl: 2}}})

        backend.advanceTo(SINCE + 3_000)

        expect(changes.map(rows)).toEqual([{countries: {pl: 0, de: 2}, landmasses: {1: {de: 1}, 2: {de: 1}}}])
        expect(rows(await backend.getHoldings())).toEqual({countries: {de: 2}, landmasses: {1: {de: 1}, 2: {de: 1}}})
    })

    it("says what each struck tile kept, from its own map when the recording is older than the server saying it", () => {
        const backend = replay(bomb(1, [], [{tileId: 2, shields: 1}]), new ReplayedEvent({
            at: at(2),
            event: new PlanetEvent({event: {case: "bombDropped", value: new BombDropped({tileId: 2, struckTileIds: [2]})}}),
        }))
        const struck: {tile: number, shields: number}[][] = []
        backend.listenForBombs((drop: BombDrop) => struck.push(drop.struck))

        backend.advanceTo(SINCE + 2_000)

        expect(struck).toEqual([[{tile: 2, shields: 1}], [{tile: 2, shields: 0}]])
    })

    it("hands over what happened up to a time, once, the updates batched between the rest", () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [2]), update(3, 3, "", "de"), spread(30))
        const seen: string[] = []
        backend.listenForUpdatesBatch((updates: Update[]) => seen.push(`updates ${updates.map((u) => u.tile).join(",")}`))
        backend.listenForBombs((drop: BombDrop) => seen.push(`bomb ${drop.cleared.join(",")}`))
        backend.listenForBonuses({
            onSpread: (click: SpreadClick) => seen.push(`spread ${click.tile}`),
            onOffered: () => {}, onTaken: () => {}, onEnclosed: () => {}, onCharges: () => {}, onRules: () => {},
        })

        backend.advanceTo(SINCE + 3_000)
        backend.advanceTo(SINCE + 3_000)
        backend.advanceTo(SINCE + 60_000)

        expect(seen).toEqual(["updates 1", "bomb 2", "updates 3", "spread 1"])
    })

    it("lists every change of hands, a bombed tile going to nobody", () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [1, 2]), update(3, 3, "", "de"))

        expect(backend.changes()).toEqual([
            {tile: 1, from: "pl", to: "de", at: SINCE + 1_000},
            {tile: 1, from: "de", to: undefined, at: SINCE + 2_000},
            {tile: 2, from: "pl", to: undefined, at: SINCE + 2_000},
            {tile: 3, from: undefined, to: "de", at: SINCE + 3_000},
        ])
    })

    it("lists its bombs, with when each fell", () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [1, 2]))

        expect(backend.drops().map(({at, drop}) => [at, drop.tile, drop.cleared])).toEqual([[SINCE + 2_000, 1, [1, 2]]])
    })

    it("lists its spreads and its shapes closed, each on its tile", () => {
        const enclosed = new ReplayedEvent({
            at: at(4),
            event: new PlanetEvent({event: {case: "tilesEnclosed", value: new TilesEnclosed({
                countryId: "fr", closingTileId: 7, wallTileIds: [7, 8], filledTileIds: [9],
            })}}),
        })
        const backend = replay(update(1, 1, "pl", "de"), spread(2), bomb(3, [1]), enclosed)

        expect(backend.effects()).toEqual([
            {at: SINCE + 2_000, kind: "spread", tile: 1},
            {at: SINCE + 4_000, kind: "enclosure", tile: 7},
        ])
    })

    it("cuts a window that opens on the map as it was at its start", async () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [2]), update(3, 3, "", "de")).cut(SINCE + 2_500, SINCE + 60_000)

        const [read] = await readOf(backend)

        expect(read.bindings).toEqual(new Map([[1, "de"]]))
        expect(read.version.seq).toBe(0)
        expect(backend.since).toBe(SINCE + 2_500)
        expect(backend.changes()).toEqual([{tile: 3, from: undefined, to: "de", at: SINCE + 3_000}])
    })
})
