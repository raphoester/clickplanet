import {STANDINGS_SHOWN} from "../domain/standings.ts"
import {TileClicker} from "./backend.ts"
import {NameColor, PlayerTitle} from "./player.ts"
import {MySeason, Race, Standing, StandingsBackend} from "./standings.ts"

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
const MAP_TILES = 262_119
const DAY_ENDS_AT_UTC_HOUR = 21
const DAY_MS = 24 * 60 * 60 * 1000

export const MOVE_EVERY_MS = 1_500
const MOST_PER_MOVE = 40

export class FakeStandingsBackend implements StandingsBackend {
    private readonly taken = new Map<string, number>()
    private readonly players: FakePlayer[]
    private readonly listeners = new Set<() => void>()
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
            .map(({tiles, ...player}) => ({...player, ...lineOf(new Map(Object.entries(tiles)), countryCode)}))
            .filter((line) => line.tiles > 0)
            .sort((a, b) => b.tiles - a.tiles)
        return sorted
            .map((line) => ({...line, rank: sorted.findIndex((other) => other.tiles === line.tiles) + 1}))
            .slice(0, STANDINGS_SHOWN)
    }

    private race(): Race {
        const held = new Map(this.taken)
        for (const player of this.players) {
            for (const [code, tiles] of Object.entries(player.tiles)) held.set(code, (held.get(code) ?? 0) + tiles)
        }
        const standings = [...held]
            .sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
            .map(([countryCode, tiles], i) => ({rank: i + 1, countryCode, share: tiles / MAP_TILES, points: DAY_POINTS[i] ?? 0}))
        const scores = Object.entries(SEASON_POINTS)
            .map(([countryCode, [points, roundsWon]], i) => ({rank: i + 1, countryCode, points, roundsWon}))

        return {round: {number: 5, endsAt: dayEndAfter(this.now()), finale: false, standings}, scores}
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
