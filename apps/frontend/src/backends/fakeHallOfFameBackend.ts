import {fakeAccountId} from "./fakeAccounts.ts"
import {SEASON_ZERO} from "./fakeSeasonBackend.ts"
import {EndedSeason, HallOfFameBackend} from "./hallOfFame.ts"
import {NameColor} from "./player.ts"
import {Standing} from "./standings.ts"

const player = (rank: number, name: string, color: NameColor, countryCode: string, tiles: number): Standing =>
    ({rank, accountId: fakeAccountId(name), name, color, countryCode, tiles})

export const sampleEndedSeason = (): EndedSeason => ({
    number: SEASON_ZERO.number,
    endedAt: SEASON_ZERO.endsAt,
    scores: [
        {rank: 1, countryCode: "fr", points: 214, roundsWon: 4},
        {rank: 2, countryCode: "br", points: 187, roundsWon: 3},
        {rank: 3, countryCode: "in", points: 140, roundsWon: 2},
        {rank: 4, countryCode: "nz", points: 96, roundsWon: 1},
        {rank: 5, countryCode: "es", points: 71, roundsWon: 0},
        {rank: 6, countryCode: "ru", points: 58, roundsWon: 0},
        {rank: 7, countryCode: "gh", points: 40, roundsWon: 0},
        {rank: 8, countryCode: "jp", points: 33, roundsWon: 0},
        {rank: 9, countryCode: "tr", points: 21, roundsWon: 0},
        {rank: 10, countryCode: "ae", points: 12, roundsWon: 0},
        {rank: 11, countryCode: "de", points: 8, roundsWon: 0},
        {rank: 12, countryCode: "us", points: 3, roundsWon: 0},
    ],
    standings: [
        {...player(1, "Ana", NameColor.PINK, "fr", 9_840), wornTitle: {id: "og", name: "OG"}},
        player(2, "Mateus", NameColor.TEAL, "br", 8_112),
        player(3, "kiran_07", NameColor.UNSPECIFIED, "in", 6_015),
        player(4, "zoe_nz", NameColor.VIOLET, "nz", 4_407),
        player(5, "Jean Moulin", NameColor.BLUE, "fr", 3_990),
        player(5, "Ольга", NameColor.RED, "ru", 3_990),
        player(7, "Kofi", NameColor.GREEN, "gh", 2_731),
        player(8, "Lucía", NameColor.ORANGE, "es", 2_655),
        player(9, "Hana", NameColor.CYAN, "jp", 1_590),
        player(10, "Bastien", NameColor.YELLOW, "fr", 1_402),
    ],
})

export class FakeHallOfFameBackend implements HallOfFameBackend {
    constructor(private readonly seasons: EndedSeason[]) {
    }

    public async hallOfFame(): Promise<EndedSeason[]> {
        return this.seasons
    }
}
