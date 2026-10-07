import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {GetReplayResponse, ReplayedEvent} from "../gen/grpc/planet/v1/admin_pb.ts"
import {BombDropped, GetMapResponse, PlanetEvent, TileUpdate, TilesSpread} from "../gen/grpc/planet/v1/planet_pb.ts"
import {BombDrop, SpreadClick, Update} from "./backend.ts"
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

function bomb(seconds: number, cleared: number[]): ReplayedEvent {
    return new ReplayedEvent({
        at: at(seconds),
        event: new PlanetEvent({event: {case: "bombDropped", value: new BombDropped({
            tileId: cleared[0], countryId: "fr", radius: 0.03, clearedTileIds: cleared,
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

function replay(...events: ReplayedEvent[]): ReplayBackend {
    return ReplayBackend.of(new GetReplayResponse({
        opening: new GetMapResponse({startTileId: 1, codes: ["", "pl"], tiles: new Uint8Array([1, 0, 1, 0, 0, 0])}),
        events,
        since: at(0),
        until: at(60),
    }))
}

describe("a replay", () => {
    it("opens on its map and says its window", async () => {
        const backend = replay()
        const loaded: Map<number, string>[] = []

        await backend.getCurrentOwnershipsByBatch(10_000, 3, ({bindings}) => loaded.push(bindings))

        expect(loaded).toEqual([new Map([[1, "pl"], [2, "pl"]])])
        expect(backend.since).toBe(SINCE)
        expect(backend.until).toBe(SINCE + 60_000)
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

    it("cuts a window that opens on the map as it was at its start", async () => {
        const backend = replay(update(1, 1, "pl", "de"), bomb(2, [2]), update(3, 3, "", "de")).cut(SINCE + 2_500, SINCE + 60_000)
        const loaded: Map<number, string>[] = []

        await backend.getCurrentOwnershipsByBatch(10_000, 3, ({bindings}) => loaded.push(bindings))

        expect(loaded).toEqual([new Map([[1, "de"]])])
        expect(backend.since).toBe(SINCE + 2_500)
        expect(backend.changes()).toEqual([{tile: 3, from: undefined, to: "de", at: SINCE + 3_000}])
    })
})
