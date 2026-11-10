import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {PlanetEvent, TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts"
import {
    BombDrop,
    Bomber,
    BonusHandlers,
    BonusListener,
    BonusLostError,
    ClaimedBonus,
    Counts,
    MapVersion,
    Ownerships,
    OwnershipsGetter,
    TileRange,
    Update,
    UpdatesListener,
} from "./backend.ts"
import {bindingsOf, bombOf, enclosureOf, shieldsOf, spreadOf, takenOf, updateOf} from "./planetBackend.ts"
import {TileChange} from "../domain/clip/changes.ts"
import {FakeTally} from "./fakeTally.ts"
import {MapHistory, MapReel} from "../domain/mapHistory.ts"

type Step = {at: number, event: PlanetEvent}

export type ReplayLandmasses = {assignment: Uint16Array, count: number}

type Reel = {
    since: number
    until: number
    opening: ReadonlyMap<number, string>
    shields: ReadonlyMap<number, number>
    steps: readonly Step[]
    landmasses: ReplayLandmasses
}

// A replay counts its writes from its opening, as one start of the server.
const EPOCH = 1

// Plays a replay as the live stream would have: advanceTo hands over everything that happened up to a time.
export class ReplayBackend implements OwnershipsGetter, UpdatesListener, BonusListener, Bomber {
    readonly since: number
    readonly until: number
    readonly opening: ReadonlyMap<number, string>
    private readonly openingShields: ReadonlyMap<number, number>
    private readonly steps: readonly Step[]
    private readonly landmasses: ReplayLandmasses
    private readonly owners: Map<number, string>
    private readonly shields: Map<number, number>
    private readonly tally: FakeTally
    private played = 0
    private readonly updateListeners = new Set<(update: Update) => void>()
    private readonly batchListeners = new Set<(updates: Update[]) => void>()
    private readonly bombListeners = new Set<(drop: BombDrop) => void>()
    private readonly holdingsListeners = new Set<(change: Counts) => void>()
    private readonly bonusHandlers = new Set<BonusHandlers>()

    static of(response: GetReplayResponse, landmasses: ReplayLandmasses): ReplayBackend {
        const opening = response.opening
        if (!opening || !response.since || !response.until) throw new Error("a replay needs its opening and its window")

        return new ReplayBackend({
            since: response.since.toDate().getTime(),
            until: response.until.toDate().getTime(),
            opening: bindingsOf(opening),
            shields: shieldsOf(opening),
            steps: response.events.flatMap(({at, event}) => at && event ? [{at: at.toDate().getTime(), event}] : []),
            landmasses,
        })
    }

    // A map history played from its frame at or before since to its end: each change at the time it was made, its last
    // of the step, so an enclosure's tiles land together and clicks one by one, as they did.
    static ofHistory(history: MapHistory, reel: MapReel, since: number, landmasses: ReplayLandmasses): ReplayBackend {
        const frames = history.frames
        if (frames.length === 0) throw new Error("a history needs a frame")
        let first = 0
        while (first + 1 < frames.length && frames[first + 1].at <= since) first++

        const owners = new Map<number, string>()
        reel.ownersAt(first).forEach((owner, tile) => {
            if (owner !== 0) owners.set(tile, reel.codes[owner])
        })
        const opening = new Map(owners)
        const steps: Step[] = []
        for (let k = first + 1; k < frames.length; k++) {
            const {tiles, owners: next, changedAt} = frames[k]
            const moved = Array.from(tiles, (tile, i) => ({tile, owner: next[i], at: Math.min(frames[k].at, Math.max(frames[k - 1].at, changedAt[i]))}))
                .filter(({tile, owner}) => (owners.get(tile) ?? "") !== owner)
                .sort((a, b) => a.at - b.at || a.tile - b.tile)
            for (const {tile, owner, at} of moved) {
                const previous = owners.get(tile) ?? ""
                if (owner === "") owners.delete(tile)
                else owners.set(tile, owner)
                const event = new PlanetEvent({
                    event: {case: "tileUpdate", value: new TileUpdate({tileId: tile, countryId: owner, previousCountryId: previous})},
                })
                steps.push({at, event})
            }
        }

        return new ReplayBackend({
            since: frames[first].at, until: frames[frames.length - 1].at, opening, shields: new Map(), steps, landmasses,
        })
    }

    private constructor(reel: Reel) {
        this.since = reel.since
        this.until = reel.until
        this.opening = reel.opening
        this.openingShields = reel.shields
        this.steps = reel.steps
        this.landmasses = reel.landmasses
        this.owners = new Map(reel.opening)
        this.shields = new Map(reel.shields)
        this.tally = new FakeTally(reel.landmasses.assignment, reel.landmasses.count)
        this.owners.forEach((owner, tile) => this.tally.moved(tile, undefined, owner))
        this.tally.changes(this.version())
    }

    // The same replay from since to until, opening on the map as it was at since.
    cut(since: number, until: number): ReplayBackend {
        const owners = new Map(this.opening)
        const shields = new Map(this.openingShields)
        let first = 0
        for (; first < this.steps.length && this.steps[first].at < since; first++) {
            const {event} = this.steps[first]
            const update = updateOf(event)
            if (update) {
                if (update.newCountry === undefined) owners.delete(update.tile)
                else owners.set(update.tile, update.newCountry)
                if (update.shields > 0) shields.set(update.tile, update.shields)
                else shields.delete(update.tile)
            }
            const bomb = bombOf(event)
            for (const tile of bomb?.cleared ?? []) {
                owners.delete(tile)
                shields.delete(tile)
            }
            const enclosure = enclosureOf(event)
            for (const tile of enclosure ? takenOf(enclosure) : []) {
                owners.set(tile, enclosure!.countryId)
                shields.delete(tile)
            }
            for (const {tile, shields: left} of [...bomb?.struck ?? [], ...enclosure?.struck ?? []]) {
                if (left > 0) shields.set(tile, left)
                else shields.delete(tile)
            }
        }

        return new ReplayBackend({
            since,
            until,
            opening: owners,
            shields,
            steps: this.steps.slice(first).filter((step) => step.at <= until),
            landmasses: this.landmasses,
        })
    }

    changes(): TileChange[] {
        const owners = new Map(this.opening)
        const changes: TileChange[] = []
        const change = (tile: number, to: string | undefined, at: number) => {
            const from = owners.get(tile)
            if (from === to) return
            changes.push({tile, from, to, at})
            if (to === undefined) owners.delete(tile)
            else owners.set(tile, to)
        }

        for (const {at, event} of this.steps) {
            const update = updateOf(event)
            if (update) change(update.tile, update.newCountry, at)
            for (const tile of bombOf(event)?.cleared ?? []) change(tile, undefined, at)
            const enclosure = enclosureOf(event)
            for (const tile of enclosure ? takenOf(enclosure) : []) change(tile, enclosure!.countryId, at)
        }

        return changes
    }

    drops(): {at: number, drop: BombDrop}[] {
        return this.steps.flatMap(({at, event}) => {
            const drop = bombOf(event)
            return drop ? [{at, drop}] : []
        })
    }

    // The other effects it plays, each on the tile it went off at: the tile a spread clicked, the tile that closed a shape.
    effects(): {at: number, kind: "spread" | "enclosure", tile: number}[] {
        return this.steps.flatMap(({at, event}): {at: number, kind: "spread" | "enclosure", tile: number}[] => {
            const spread = spreadOf(event)
            if (spread) return [{at, kind: "spread", tile: spread.tile}]
            const enclosure = enclosureOf(event)
            return enclosure ? [{at, kind: "enclosure", tile: enclosure.closingTile}] : []
        })
    }

    advanceTo(at: number) {
        let updates: Update[] = []
        const flush = () => {
            if (updates.length === 0) return
            for (const listener of this.batchListeners) listener(updates)
            updates = []
        }

        while (this.played < this.steps.length && this.steps[this.played].at <= at) {
            const {event} = this.steps[this.played++]
            const version = this.version()

            const read = updateOf(event)
            if (read) {
                const update = {...read, version}
                this.own(update.tile, update.newCountry, update.shields)
                updates.push(update)
                for (const listener of this.updateListeners) listener(update)
                continue
            }

            flush()
            const dropped = bombOf(event)
            if (dropped) {
                const bomb = {...dropped, version}
                for (const tile of bomb.cleared) this.own(tile, undefined, 0)
                for (const {tile, shields} of bomb.struck) this.own(tile, this.owners.get(tile), shields)
                for (const listener of this.bombListeners) listener(bomb)
            }
            const spread = spreadOf(event)
            if (spread) for (const handlers of this.bonusHandlers) handlers.onSpread(spread)
            const enclosed = enclosureOf(event)
            if (enclosed) {
                const enclosure = {...enclosed, version}
                for (const tile of takenOf(enclosure)) this.own(tile, enclosure.countryId, 0)
                for (const {tile, shields} of enclosure.struck) this.own(tile, this.owners.get(tile), shields)
                for (const handlers of this.bonusHandlers) handlers.onEnclosed(enclosure)
            }
        }

        flush()
        const change = this.tally.changes(this.version())
        if (change) for (const listener of this.holdingsListeners) listener(change)
    }

    async getFortresses(): Promise<Map<number, string>> {
        return new Map()
    }

    async getHoldings(): Promise<Counts> {
        return this.tally.read(this.version())
    }

    async getOwnerships(ranges: readonly TileRange[], callback: (ownerships: Ownerships) => void): Promise<void> {
        for (const range of ranges) {
            const bindings = new Map<number, string>()
            const shields = new Map<number, number>()
            for (let tile = range.start; tile <= range.end; tile++) {
                const owner = this.owners.get(tile)
                if (owner !== undefined) bindings.set(tile, owner)
                const held = this.shields.get(tile)
                if (held) shields.set(tile, held)
            }
            callback({range, version: this.version(), bindings, shields, arena: false, knockedOut: []})
        }
    }

    listenForUpdates(callback: (update: Update) => void): () => void {
        this.updateListeners.add(callback)
        return () => this.updateListeners.delete(callback)
    }

    listenForUpdatesBatch(callback: (updates: Update[]) => void): () => void {
        this.batchListeners.add(callback)
        return () => this.batchListeners.delete(callback)
    }

    listenForOpenings(callback: (version: MapVersion) => void): () => void {
        callback(this.version())
        return () => {}
    }

    listenForHoldings(callback: (change: Counts) => void): () => void {
        this.holdingsListeners.add(callback)
        return () => this.holdingsListeners.delete(callback)
    }

    listenForFortifications(): () => void {
        return () => {}
    }

    listenForKnockOuts(): () => void {
        return () => {}
    }

    listenForBonuses(handlers: BonusHandlers): () => void {
        this.bonusHandlers.add(handlers)
        return () => this.bonusHandlers.delete(handlers)
    }

    claimBonus(): Promise<ClaimedBonus> {
        return Promise.reject(new BonusLostError())
    }

    listenForBombs(onDropped: (drop: BombDrop) => void): () => void {
        this.bombListeners.add(onDropped)
        return () => this.bombListeners.delete(onDropped)
    }

    dropBomb(): Promise<void> {
        return Promise.reject(new Error("a replay drops no bomb"))
    }

    private version(): MapVersion {
        return {epoch: EPOCH, seq: this.played}
    }

    private own(tile: number, owner: string | undefined, shields: number) {
        this.tally.moved(tile, this.owners.get(tile), owner)
        if (owner === undefined) this.owners.delete(tile)
        else this.owners.set(tile, owner)
        if (shields > 0) this.shields.set(tile, shields)
        else this.shields.delete(tile)
    }
}
