import {STANDINGS_SHOWN} from "../domain/standings.ts"
import {TileClicker} from "./backend.ts"
import {NameColor, PlayerTitle} from "./player.ts"
import {MySeason, Standing, StandingsBackend} from "./standings.ts"

type FakePlayer = Pick<Standing, "name" | "color" | "wornTitle"> & {tiles: Record<string, number>}

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

export class FakeStandingsBackend implements StandingsBackend {
    private readonly taken = new Map<string, number>()

    constructor(private readonly players: readonly FakePlayer[] = PLAYERS) {
    }

    public counting(clicker: TileClicker): TileClicker {
        return {
            clickTile: async (tileId, countryId, switches) => {
                await clicker.clickTile(tileId, countryId, switches)
                this.taken.set(countryId, (this.taken.get(countryId) ?? 0) + 1)
            },
        }
    }

    public async standings(countryCode: string): Promise<Standing[]> {
        const sorted = this.players
            .map(({tiles, ...player}) => ({...player, ...lineOf(new Map(Object.entries(tiles)), countryCode)}))
            .filter((line) => line.tiles > 0)
            .sort((a, b) => b.tiles - a.tiles)
        return sorted
            .map((line) => ({...line, rank: sorted.findIndex((other) => other.tiles === line.tiles) + 1}))
            .slice(0, STANDINGS_SHOWN)
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
