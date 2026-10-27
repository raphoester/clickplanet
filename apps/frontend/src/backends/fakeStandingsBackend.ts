import {fakeAccountId} from "./fakeAccounts.ts"
import {STANDINGS_SHOWN} from "../domain/standings.ts"
import {TileClicker} from "./backend.ts"
import {NameColor, PlayerTitle} from "./player.ts"
import {ClosedRound, MySeason, Race, Round, RoundStanding, RoundTakes, Score, Standing, StandingsBackend} from "./standings.ts"

export type FakePlayer = Pick<Standing, "name" | "color" | "wornTitle"> & {tiles: Record<string, number>}

const conquest = (id: string, name: string, number: number): PlayerTitle =>
    ({id, name, rank: {trackId: "conquest", trackName: "Conquest", number, count: 5}})

const WARLORD = conquest("warlord", "Warlord", 3)
const RAIDER = conquest("raider", "Raider", 2)
const SETTLER = conquest("settler", "Settler", 1)
const DEVOTED: PlayerTitle = {id: "devoted", name: "Devoted", rank: {trackId: "devotion", trackName: "Devotion", number: 2, count: 3}}
const OG: PlayerTitle = {id: "og", name: "OG"}

const PLAYERS: FakePlayer[] = [
    {name: "Ana", color: NameColor.PINK, tiles: {fr: 1_840, es: 210}, wornTitle: WARLORD},
    {name: "kiran_07", color: NameColor.UNSPECIFIED, tiles: {in: 1_512}},
    {name: "Mateus", color: NameColor.TEAL, tiles: {br: 1_512, fr: 75}, wornTitle: OG},
    {name: "zoe_nz", color: NameColor.VIOLET, tiles: {nz: 1_207}, wornTitle: DEVOTED},
    {name: "Jean Moulin", color: NameColor.BLUE, tiles: {fr: 990}},
    {name: "Ольга", color: NameColor.RED, tiles: {ru: 864, fr: 120}},
    {name: "Kofi", color: NameColor.GREEN, tiles: {gh: 731}, wornTitle: RAIDER},
    {name: "Lucía", color: NameColor.ORANGE, tiles: {es: 655, fr: 31}},
    {name: "Hana", color: NameColor.CYAN, tiles: {jp: 590}},
    {name: "Bastien", color: NameColor.YELLOW, tiles: {fr: 402}, wornTitle: SETTLER},
    {name: "Mehmet", color: NameColor.MAGENTA, tiles: {tr: 377}},
    {name: "Inès", color: NameColor.LIME, tiles: {fr: 215}, wornTitle: RAIDER},
    {name: "Pierre_L", color: NameColor.INDIGO, tiles: {fr: 96}},
    {name: "Noor", color: NameColor.UNSPECIFIED, tiles: {ae: 44, in: 12}},
]

const SEASON_POINTS: Record<string, [points: number, roundsWon: number]> = {
    fr: [61, 2], br: [43, 1], in: [36, 1], nz: [25, 1], es: [18, 0], ru: [12, 0],
}
const DAY_POINTS = [25, 18, 15, 12, 10, 8, 6, 4, 2, 1]
const FIRST_DAY = 5
const PLAYERS_SHOWN = 10
// The part of what a flag holds its players took during a battle.
const TAKEN_IN_BATTLE = 0.2
const MAP_TILES = 262_119
const DAY_ENDS_AT_UTC_HOUR = 21
const DAY_MS = 24 * 60 * 60 * 1000

export const MOVE_EVERY_MS = 1_500
const MOST_PER_MOVE = 40

export class FakeStandingsBackend implements StandingsBackend {
    private readonly taken = new Map<string, number>()
    private readonly players: FakePlayer[]
    private readonly listeners = new Set<() => void>()
    private readonly season = new Map(Object.entries(SEASON_POINTS).map(([code, [points, roundsWon]]) => [code, {points, roundsWon}]))
    private day = FIRST_DAY
    private battles = 0
    private battle?: {entryId: string, since: number, endsAt: number, times: number, taken: Map<string, number>}
    private closed?: ClosedRound
    private timer?: ReturnType<typeof setInterval>

    constructor(
        players: readonly FakePlayer[] = PLAYERS,
        private readonly random: () => number = Math.random,
        private readonly now: () => number = Date.now,
    ) {
        this.players = players.map((player) => ({...player, tiles: {...player.tiles}}))
    }

    public counting(clicker: TileClicker): TileClicker {
        return {
            clickTile: async (tileId, countryId, switches) => {
                await clicker.clickTile(tileId, countryId, switches)
                this.taken.set(countryId, (this.taken.get(countryId) ?? 0) + 1)
            },
        }
    }

    public listenForStandings(countryCode: string, onStandings: (standings: Standing[]) => void): () => void {
        return this.listen(() => onStandings(this.top(countryCode)))
    }

    public listenForRace(onRace: (race: Race) => void): () => void {
        return this.listen(() => onRace(this.race()))
    }

    private listen(send: () => void): () => void {
        this.listeners.add(send)
        void Promise.resolve().then(() => {
            if (this.listeners.has(send)) send()
        })
        this.timer ??= setInterval(() => this.move(), MOVE_EVERY_MS)

        return () => {
            this.listeners.delete(send)
            if (this.listeners.size > 0) return
            clearInterval(this.timer)
            this.timer = undefined
        }
    }

    private move() {
        const player = this.players[Math.floor(this.random() * this.players.length)]
        const flags = Object.keys(player.tiles)
        const flag = flags[Math.floor(this.random() * flags.length)]
        player.tiles[flag] += 1 + Math.floor(this.random() * MOST_PER_MOVE)
        this.listeners.forEach((send) => send())
    }

    private top(countryCode: string): Standing[] {
        const sorted = this.players
            .map(({tiles, ...player}) => ({
                ...player,
                accountId: fakeAccountId(player.name),
                ...lineOf(new Map(Object.entries(tiles)), countryCode),
            }))
            .filter((line) => line.tiles > 0)
            .sort((a, b) => b.tiles - a.tiles)
        return sorted
            .map((line) => ({...line, rank: sorted.findIndex((other) => other.tiles === line.tiles) + 1}))
            .slice(0, STANDINGS_SHOWN)
    }

    public closeRound(): ClosedRound {
        const before = this.table()
        const standings = this.scored(this.standings(1, () => 0.4 + this.random()))
        const closed: ClosedRound = {
            season: 0, number: this.day, endedAt: this.now(), finale: false, standings, before, after: this.table(),
            own: false, times: 1, startedAt: 0, entryId: "", takes: [], players: [],
        }
        this.day += 1
        return this.close(closed)
    }

    public openBattle(entryId: string, since: number, endsAt: number, times: number) {
        this.battle = {entryId, since, endsAt, times, taken: new Map(this.taken)}
        this.listeners.forEach((send) => send())
    }

    // The battle opened, else one of 2 hours at ×3 that ends now.
    public closeBattle(finale = false): ClosedRound {
        const now = this.now()
        const battle = this.battle ?? {
            entryId: crypto.randomUUID(), since: now - 2 * 60 * 60 * 1000, endsAt: now, times: 3, taken: new Map(this.taken),
        }
        const before = this.table()
        const standings = this.scored(this.standings(battle.times, () => 0.4 + this.random()))
        this.battles += 1
        this.battle = undefined
        return this.close({
            season: 0, number: this.battles, endedAt: Math.min(now, battle.endsAt), finale, standings, before, after: this.table(),
            own: true, times: battle.times, startedAt: battle.since, entryId: battle.entryId, takes: this.takes(battle.taken),
            players: this.fighters(),
        })
    }

    private close(closed: ClosedRound): ClosedRound {
        this.closed = closed
        this.listeners.forEach((send) => send())
        return closed
    }

    private scored(standings: RoundStanding[]): RoundStanding[] {
        for (const standing of standings) {
            const kept = this.season.get(standing.countryCode) ?? {points: 0, roundsWon: 0}
            this.season.set(standing.countryCode, {
                points: kept.points + standing.points,
                roundsWon: kept.roundsWon + (standing.rank === 1 ? 1 : 0),
            })
        }
        return standings.filter((standing) => standing.points > 0)
    }

    private takes(takenBefore: ReadonlyMap<string, number>): RoundTakes[] {
        const takes = new Map<string, number>()
        for (const [code, tiles] of this.taken) takes.set(code, tiles - (takenBefore.get(code) ?? 0))
        for (const player of this.players) {
            for (const [code, tiles] of Object.entries(player.tiles)) {
                takes.set(code, (takes.get(code) ?? 0) + Math.round(tiles * TAKEN_IN_BATTLE * this.random()))
            }
        }
        return [...takes]
            .filter(([, tiles]) => tiles > 0)
            .sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
            .map(([countryCode, tiles]) => ({countryCode, tiles}))
    }

    private fighters(): Standing[] {
        const sorted = this.players
            .map(({tiles, ...player}) => {
                const [countryCode, held] = Object.entries(tiles).sort(([, x], [, y]) => y - x)[0]
                return {...player, accountId: fakeAccountId(player.name), countryCode, tiles: Math.round(held * TAKEN_IN_BATTLE * this.random())}
            })
            .filter((line) => line.tiles > 0)
            .sort((a, b) => b.tiles - a.tiles)
        return sorted
            .map((line) => ({...line, rank: sorted.findIndex((other) => other.tiles === line.tiles) + 1}))
            .slice(0, PLAYERS_SHOWN)
    }

    private race(): Race {
        return {round: this.round(), scores: this.table(), closed: this.closed}
    }

    private round(): Round {
        if (this.battle) {
            const {entryId, endsAt, times} = this.battle
            return {number: this.battles + 1, endsAt, finale: false, standings: this.standings(times, () => 1), own: true, times, entryId}
        }
        return {number: this.day, endsAt: dayEndAfter(this.now()), finale: false, standings: this.standings(1, () => 1), own: false, times: 1, entryId: ""}
    }

    private standings(times: number, luck: () => number): RoundStanding[] {
        const held = new Map(this.taken)
        for (const player of this.players) {
            for (const [code, tiles] of Object.entries(player.tiles)) held.set(code, (held.get(code) ?? 0) + tiles)
        }
        return [...held]
            .map(([code, tiles]) => [code, tiles * luck()] as const)
            .sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
            .map(([countryCode, tiles], i) => ({
                rank: i + 1,
                countryCode,
                share: tiles / MAP_TILES,
                points: (DAY_POINTS[i] ?? 0) * times,
            }))
    }

    private table(): Score[] {
        const sorted = [...this.season]
            .filter(([, kept]) => kept.points > 0)
            .map(([countryCode, kept]) => ({countryCode, ...kept}))
            .sort((a, b) => b.points - a.points || b.roundsWon - a.roundsWon || a.countryCode.localeCompare(b.countryCode))
        return sorted.map((score) => ({
            ...score,
            rank: sorted.findIndex((other) => other.points === score.points && other.roundsWon === score.roundsWon) + 1,
        }))
    }

    public async mySeason(countryCode: string): Promise<MySeason | undefined> {
        const line = lineOf(this.taken, countryCode)
        return {countryCode: line.countryCode || undefined, tiles: line.tiles}
    }
}

function lineOf(tiles: ReadonlyMap<string, number>, countryCode: string): {countryCode: string, tiles: number} {
    if (countryCode !== "") return {countryCode, tiles: tiles.get(countryCode) ?? 0}

    let main = {countryCode: "", tiles: 0}
    for (const [code, count] of tiles) {
        if (count > main.tiles) main = {countryCode: code, tiles: count}
    }
    return main
}

function dayEndAfter(now: number): number {
    const end = new Date(now)
    end.setUTCHours(DAY_ENDS_AT_UTC_HOUR, 0, 0, 0)
    return end.getTime() > now ? end.getTime() : end.getTime() + DAY_MS
}
