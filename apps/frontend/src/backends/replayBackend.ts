import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {PlanetEvent} from "../gen/grpc/planet/v1/planet_pb.ts"
import {
    BombDrop,
    Bomber,
    BonusHandlers,
    BonusListener,
    BonusLostError,
    ClaimedBonus,
    Ownerships,
    OwnershipsGetter,
    Update,
    UpdatesListener,
} from "./backend.ts"
import {bindingsOf, bombOf, enclosureOf, shieldsOf, spreadOf, updateOf} from "./planetBackend.ts"
import {TileChange} from "../domain/clip/changes.ts"

type Step = {at: number, event: PlanetEvent}

type Reel = {
    since: number
    until: number
    opening: ReadonlyMap<number, string>
    shields: ReadonlyMap<number, number>
    steps: readonly Step[]
}

// Plays a replay as the live stream would have: advanceTo hands over everything that happened up to a time.
export class ReplayBackend implements OwnershipsGetter, UpdatesListener, BonusListener, Bomber {
    readonly since: number
    readonly until: number
    readonly opening: ReadonlyMap<number, string>
    private readonly shields: ReadonlyMap<number, number>
    private readonly steps: readonly Step[]
    private played = 0
    private readonly updateListeners = new Set<(update: Update) => void>()
    private readonly batchListeners = new Set<(updates: Update[]) => void>()
    private readonly bombListeners = new Set<(drop: BombDrop) => void>()
    private readonly bonusHandlers = new Set<BonusHandlers>()

    static of(response: GetReplayResponse): ReplayBackend {
        const opening = response.opening
        if (!opening || !response.since || !response.until) throw new Error("a replay needs its opening and its window")

        return new ReplayBackend({
            since: response.since.toDate().getTime(),
            until: response.until.toDate().getTime(),
            opening: bindingsOf(opening),
            shields: shieldsOf(opening),
            steps: response.events.flatMap(({at, event}) => at && event ? [{at: at.toDate().getTime(), event}] : []),
        })
    }

    private constructor(reel: Reel) {
        this.since = reel.since
        this.until = reel.until
        this.opening = reel.opening
        this.shields = reel.shields
        this.steps = reel.steps
    }

    // The same replay from since to until, opening on the map as it was at since.
    cut(since: number, until: number): ReplayBackend {
        const owners = new Map(this.opening)
        const shields = new Map(this.shields)
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
            for (const tile of bomb?.struck ?? []) {
                const left = (shields.get(tile) ?? 1) - 1
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
        }

        return changes
    }

    drops(): {at: number, drop: BombDrop}[] {
        return this.steps.flatMap(({at, event}) => {
            const drop = bombOf(event)
            return drop ? [{at, drop}] : []
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

            const update = updateOf(event)
            if (update) {
                updates.push(update)
                for (const listener of this.updateListeners) listener(update)
                continue
            }

            flush()
            const bomb = bombOf(event)
            if (bomb) for (const listener of this.bombListeners) listener(bomb)
            const spread = spreadOf(event)
            if (spread) for (const handlers of this.bonusHandlers) handlers.onSpread(spread)
            const enclosure = enclosureOf(event)
            if (enclosure) for (const handlers of this.bonusHandlers) handlers.onEnclosed(enclosure)
        }

        flush()
    }

    async getFortresses(): Promise<Map<number, string>> {
        return new Map()
    }

    async getCurrentOwnershipsByBatch(
        _batchSize: number,
        _maxIndex: number,
        callback: (ownerships: Ownerships) => void,
    ): Promise<void> {
        callback({bindings: new Map(this.opening), shields: new Map(this.shields)})
    }

    listenForUpdates(callback: (update: Update) => void): () => void {
        this.updateListeners.add(callback)
        return () => this.updateListeners.delete(callback)
    }

    listenForUpdatesBatch(callback: (updates: Update[]) => void): () => void {
        this.batchListeners.add(callback)
        return () => this.batchListeners.delete(callback)
    }

    listenForResumes(): () => void {
        return () => {}
    }

    listenForFortifications(): () => void {
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
}
