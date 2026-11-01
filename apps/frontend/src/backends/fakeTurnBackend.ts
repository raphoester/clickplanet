import {Day, Season, Seasons} from "./season.ts"
import {BattleEnded, BattleToCome, Debrief, PLAIN_TURN, Turn, TurnBackend, TurnListener} from "./turn.ts"

const MINUTE = 60_000
const DAY_MS = 24 * 60 * MINUTE
const CUTOFF_UTC_HOUR = 21

export const SEASON_ZERO: Season = {
    number: 0,
    startsAt: Date.UTC(2026, 9, 8),
    endsAt: Date.UTC(2026, 9, 31, 23),
    bookId: "season-0",
}

export class FakeTurnBackend implements TurnBackend {
    private current: Turn = PLAIN_TURN
    private battles: BattleToCome[] = []
    private seasons: Seasons
    private debriefed?: Debrief
    private made = 0
    private battlesEnded = 0
    private readonly listeners = new Set<TurnListener>()

    constructor(current: Season | undefined = SEASON_ZERO, private readonly now: () => number = Date.now, ended: Season[] = []) {
        this.seasons = {current, today: current && dayOf(current, now()), ended}
    }

    public follow(listener: TurnListener): () => void {
        this.listeners.add(listener)
        listener.onTurn(this.current)
        listener.onBattles(this.battles)
        listener.onSeasons(this.seasons)
        listener.onDebrief(this.debriefed)
        return () => {
            this.listeners.delete(listener)
        }
    }

    public today(): Day | undefined {
        return this.seasons.today
    }

    public startBattle(entryId: string, since: number, endsAt: number, description = "Triple points. A bonus box every minute.", times = 3): void {
        this.current = {kind: "battle", since, endsAt, description, entryId, roundId: entryId, times, endsSeason: false}
        for (const listener of this.listeners) listener.onTurn(this.current)
    }

    // The season's finale plays: no entry's, named by its round, and it ends with the season.
    public startFinale(since: number, description = "Triple points.", times = 3): string {
        const current = this.seasons.current
        const roundId = `${current?.bookId ?? "season-0"}/finale`
        this.setBattles(this.battles.filter((battle) => !battle.endsSeason))
        this.current = {
            kind: "battle", since, endsAt: current?.endsAt ?? since, description, entryId: "", roundId, times, endsSeason: true,
        }
        for (const listener of this.listeners) listener.onTurn(this.current)
        return roundId
    }

    public startHalloween(entryId: string, since: number, endsAt: number): void {
        this.current = {...PLAIN_TURN, kind: "halloween", since, endsAt, entryId}
        for (const listener of this.listeners) listener.onTurn(this.current)
    }

    public endBattle(): void {
        this.current = PLAIN_TURN
        for (const listener of this.listeners) listener.onTurn(this.current)
    }

    // Scoring scored today's round: the gamemaster tells of it, and the next day runs.
    public endDay(): Debrief | undefined {
        const {current, today} = this.seasons
        if (!current || !today) return undefined
        const debrief: Debrief = {kind: "day", season: current.number, number: today.number, roundId: today.roundId, endedAt: this.now()}
        const next = today.number + 1
        this.seasons = {...this.seasons, today: {...today, number: next, startsAt: this.now(), roundId: dayRoundOf(current, next)}}
        for (const listener of this.listeners) listener.onSeasons(this.seasons)
        this.debrief(debrief)
        return debrief
    }

    public endBattleRound(roundId: string, times: number, endedAt: number, endsSeason: boolean): BattleEnded {
        const debrief: BattleEnded = {
            kind: "battle",
            season: this.seasons.current?.number ?? 0,
            number: ++this.battlesEnded,
            entryId: endsSeason ? "" : roundId,
            times,
            endsSeason,
            roundId,
            endedAt,
        }
        this.debrief(debrief)
        return debrief
    }

    public endSeason(finaleRoundId: string, finaleBattle?: BattleEnded): Debrief | undefined {
        const current = this.seasons.current
        if (!current) return undefined
        const debrief: Debrief = {
            kind: "season", season: current.number, bookId: current.bookId, endedAt: this.now(), finaleRoundId, finaleBattle,
        }
        this.debrief(debrief)
        return debrief
    }

    public scheduleBattle(startsAt: number, minutes: number, description = "Triple points."): string {
        const entryId = `battle-${++this.made}`
        this.setBattles([...this.battles, {
            entryId, startsAt, endsAt: startsAt + minutes * MINUTE, description, times: 3, endsSeason: false, roundId: entryId,
        }])
        return entryId
    }

    // The season's finale, its length before the season's end.
    public scheduleFinale(minutes: number, description = "Triple points."): string {
        const current = this.seasons.current
        if (!current) return ""
        const roundId = `${current.bookId}/finale`
        this.setBattles([...this.battles.filter((battle) => !battle.endsSeason), {
            entryId: "", startsAt: current.endsAt - minutes * MINUTE, endsAt: current.endsAt, description, times: 3, endsSeason: true, roundId,
        }])
        return roundId
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
        const moved = {...current, endsAt}
        this.seasons = {...this.seasons, current: moved, today: dayOf(moved, this.now())}
        for (const listener of this.listeners) listener.onSeasons(this.seasons)
        if (this.battles.some((battle) => battle.endsSeason)) {
            this.setBattles(this.battles.map((battle) => battle.endsSeason
                ? {...battle, startsAt: endsAt - (battle.endsAt - battle.startsAt), endsAt}
                : battle))
        }
    }

    // The running season is over, and the next one starts later: the break between two seasons.
    public startBreak(nextStartsAt: number, days = 28): void {
        const ended = this.seasons.current
        const number = (ended?.number ?? -1) + 1
        this.seasons = {
            next: {number, startsAt: nextStartsAt, endsAt: nextStartsAt + days * DAY_MS, bookId: `season-${number}`},
            ended: ended ? [...this.seasons.ended, ended] : this.seasons.ended,
        }
        for (const listener of this.listeners) listener.onSeasons(this.seasons)
    }

    private debrief(debrief: Debrief) {
        this.debriefed = debrief
        for (const listener of this.listeners) listener.onDebrief(debrief)
    }

    private setBattles(battles: BattleToCome[]) {
        this.battles = [...battles].sort((a, b) => a.startsAt - b.startsAt)
        for (const listener of this.listeners) listener.onBattles(this.battles)
    }
}

// As the gamemaster counts: day K ends at the K-th cutoff after the season's start, and none runs after its last.
function dayOf(season: Season, now: number): Day | undefined {
    if (now < season.startsAt || now >= season.endsAt) return undefined
    const first = cutoffAfter(season.startsAt)
    const number = now < first ? 1 : 2 + Math.floor((now - first) / DAY_MS)
    const endsAt = first + (number - 1) * DAY_MS
    if (endsAt > season.endsAt) return undefined
    return {number, startsAt: number === 1 ? season.startsAt : endsAt - DAY_MS, endsAt, roundId: dayRoundOf(season, number)}
}

function cutoffAfter(at: number): number {
    const cutoff = new Date(at)
    cutoff.setUTCHours(CUTOFF_UTC_HOUR, 0, 0, 0)
    return cutoff.getTime() > at ? cutoff.getTime() : cutoff.getTime() + DAY_MS
}

function dayRoundOf(season: Season, number: number): string {
    return `${season.bookId}/day-${number}`
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
        startBreak: (nextInMinutes = 960) => {
            turns.startBreak(Date.now() + nextInMinutes * MINUTE)
            return `the season is over; the next one starts in ${nextInMinutes} minutes`
        },
        scheduleFinale: (inMinutes = 90, minutes = 120) => {
            turns.moveSeasonEnd(Date.now() + (inMinutes + minutes) * MINUTE)
            return `${turns.scheduleFinale(minutes)} starts in ${inMinutes} minutes`
        },
    }
}
