import {describe, expect, it} from "vitest"
import {FakeChatBackend} from "./fakeChatBackend.ts"
import {ClosedRound} from "./standings.ts"

const FINALE: ClosedRound = {
    season: 0,
    number: 23,
    endedAt: Date.UTC(2026, 9, 31, 23),
    finale: true,
    standings: [],
    before: [],
    after: [{rank: 1, countryCode: "de", points: 118, roundsWon: 2}, {rank: 2, countryCode: "fr", points: 97, roundsWon: 1}],
    own: true,
    times: 3,
    startedAt: Date.UTC(2026, 9, 31, 21),
    takes: [],
    players: [],
}

const championsIn = async (chat: FakeChatBackend) =>
    (await chat.getHistory()).announcements.filter((announcement) => announcement.kind === "champion")

describe("FakeChatBackend.announceChampion", () => {
    it("announces a season's champion once, however often it is told", async () => {
        const chat = new FakeChatBackend()

        chat.announceChampion(FINALE)
        chat.announceChampion({...FINALE, number: 24})

        expect(await championsIn(chat)).toEqual([{
            kind: "champion", id: "champion-0", announcedAt: FINALE.endedAt, season: 0, countries: ["de"],
        }])
    })

    it("announces nothing after a day", async () => {
        const chat = new FakeChatBackend()

        chat.announceChampion({...FINALE, finale: false})

        expect(await championsIn(chat)).toEqual([])
    })
})
