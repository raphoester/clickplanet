import {Season, Seasons} from "./season.ts"
import {BattleToCome, PLAIN_TURN, Turn, TurnBackend, TurnListener} from "./turn.ts"

const MINUTE = 60_000

export const SEASON_ZERO: Season = {
    number: 0,
    startsAt: Date.UTC(2026, 9, 3),
    endsAt: Date.UTC(2026, 9, 31, 23),
}

export class FakeTurnBackend implements TurnBackend {
    private current: Turn = PLAIN_TURN
    private battles: BattleToCome[] = []
    private seasons: Seasons
    private made = 0
    private readonly listeners = new Set<TurnListener>()

    constructor(current: Season | undefined = SEASON_ZERO) {
        this.seasons = {current}
    }

    public follow(listener: TurnListener): () => void {
        this.listeners.add(listener)
        listener.onTurn(this.current)
        listener.onBattles(this.battles)
        listener.onSeasons(this.seasons)
        return () => {
            this.listeners.delete(listener)
        }
    }

    public startBattle(entryId: string, since: number, endsAt: number, description = "Triple points. A bonus box every minute."): void {
        this.current = {kind: "battle", since, endsAt, description, entryId}
        for (const listener of this.listeners) listener.onTurn(this.current)
    }

    public endBattle(): void {
        this.current = PLAIN_TURN
        for (const listener of this.listeners) listener.onTurn(this.current)
    }

    public scheduleBattle(startsAt: number, minutes: number, description = "Triple points."): string {
        const entryId = `battle-${++this.made}`
        this.setBattles([...this.battles, {entryId, startsAt, endsAt: startsAt + minutes * MINUTE, description}])
        return entryId
    }

    public moveBattle(entryId: string, startsAt: number): void {
        this.setBattles(this.battles.map((battle) => battle.entryId === entryId
            ? {...battle, startsAt, endsAt: startsAt + battle.endsAt - battle.startsAt}
            : battle))
    }

    public deleteBattle(entryId: string): void {
        this.setBattles(this.battles.filter((battle) => battle.entryId !== entryId))
    }

    public moveSeasonEnd(endsAt: number): void {
        const current = this.seasons.current
        if (!current) return
        this.seasons = {...this.seasons, current: {...current, endsAt}}
        for (const listener of this.listeners) listener.onSeasons(this.seasons)
    }

    private setBattles(battles: BattleToCome[]) {
        this.battles = [...battles].sort((a, b) => a.startsAt - b.startsAt)
        for (const listener of this.listeners) listener.onBattles(this.battles)
    }
}

export function battleCommands(turns: FakeTurnBackend) {
    return {
        scheduleBattle: (inMinutes = 90, minutes = 120, description?: string) =>
            turns.scheduleBattle(Date.now() + inMinutes * MINUTE, minutes, description),
        moveBattle: (entryId: string, inMinutes: number) => {
            turns.moveBattle(entryId, Date.now() + inMinutes * MINUTE)
            return `${entryId} starts in ${inMinutes} minutes`
        },
        deleteBattle: (entryId: string) => {
            turns.deleteBattle(entryId)
            return `${entryId} is deleted`
        },
        moveSeasonEnd: (inMinutes: number) => {
            turns.moveSeasonEnd(Date.now() + inMinutes * MINUTE)
            return `the season ends in ${inMinutes} minutes`
        },
    }
}
