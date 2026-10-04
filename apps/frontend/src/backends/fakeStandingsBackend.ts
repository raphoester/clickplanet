import {STANDINGS_SHOWN} from "../domain/standings.ts"
import {TileClicker} from "./backend.ts"
import {NameColor} from "./player.ts"
import {MySeason, Standing, StandingsBackend} from "./standings.ts"

type FakePlayer = Omit<Standing, "rank">

const PLAYERS: FakePlayer[] = [
    {name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 1_840},
    {name: "kiran_07", color: NameColor.UNSPECIFIED, countryCode: "in", tiles: 1_512},
    {name: "Mateus", color: NameColor.TEAL, countryCode: "br", tiles: 1_512},
    {name: "zoe_nz", color: NameColor.VIOLET, countryCode: "nz", tiles: 1_207},
    {name: "Jean Moulin", color: NameColor.BLUE, countryCode: "fr", tiles: 990},
    {name: "Ольга", color: NameColor.RED, countryCode: "ru", tiles: 864},
    {name: "Kofi", color: NameColor.GREEN, countryCode: "gh", tiles: 731},
    {name: "Lucía", color: NameColor.ORANGE, countryCode: "es", tiles: 655},
    {name: "Hana", color: NameColor.CYAN, countryCode: "jp", tiles: 590},
    {name: "Bastien", color: NameColor.YELLOW, countryCode: "fr", tiles: 402},
    {name: "Mehmet", color: NameColor.MAGENTA, countryCode: "tr", tiles: 377},
    {name: "Inès", color: NameColor.LIME, countryCode: "fr", tiles: 215},
    {name: "Pierre_L", color: NameColor.INDIGO, countryCode: "fr", tiles: 96},
    {name: "Noor", color: NameColor.UNSPECIFIED, countryCode: "ae", tiles: 44},
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
            .filter((player) => countryCode === "" || player.countryCode === countryCode)
            .sort((a, b) => b.tiles - a.tiles)
        return sorted
            .map((player) => ({...player, rank: sorted.findIndex((other) => other.tiles === player.tiles) + 1}))
            .slice(0, STANDINGS_SHOWN)
    }

    public async mySeason(): Promise<MySeason | undefined> {
        let main: {countryCode: string, tiles: number} | undefined
        for (const [countryCode, tiles] of this.taken) {
            if (!main || tiles > main.tiles) main = {countryCode, tiles}
        }
        return {countryCode: main?.countryCode, tiles: main?.tiles ?? 0}
    }
}
