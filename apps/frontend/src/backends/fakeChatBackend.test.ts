import {describe, expect, it} from "vitest"
import {FakeChatBackend} from "./fakeChatBackend.ts"
import {ChatRejectedError} from "./chat.ts"
import {ClosedRound} from "./standings.ts"

const FINALE: ClosedRound = {
    season: 0,
    number: 23,
    endedAt: Date.UTC(2026, 9, 31, 23),
    finale: true,
    standings: [],
    before: [],
    after: [{rank: 1, countryCode: "de", points: 118, roundsWon: 2}, {rank: 2, countryCode: "fr", points: 97, roundsWon: 1}],
    moment: "battle",
    times: 3,
    startedAt: Date.UTC(2026, 9, 31, 21),
    entryId: "entry-final",
    roundId: "entry-final",
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

describe("FakeChatBackend.announceFlagMission", () => {
    const mission = {id: "flag", order: {kind: "breakShields", country: "dz", goal: 300, reached: 300}, points: 2, done: true} as const
    const flagMissionsIn = async (chat: FakeChatBackend) =>
        (await chat.getHistory()).announcements.filter((announcement) => announcement.kind === "flag_mission")

    it("announces it once in the World and in the flag's own room, and in no other", async () => {
        const world = new FakeChatBackend()

        world.announceFlagMission("fr", mission)
        world.flagRoom("fr").announceFlagMission("fr", mission)

        const told = {kind: "flag_mission", country: "fr", points: 2, order: mission.order}
        expect(await flagMissionsIn(world)).toMatchObject([told])
        expect(await flagMissionsIn(world.flagRoom("fr"))).toMatchObject([told])
        expect(await flagMissionsIn(world.flagRoom("de"))).toEqual([])
        world.close()
    })
})

describe("FakeChatBackend.sendMessage", () => {
    const gif = {url: "data:image/svg+xml,", width: 220, height: 124}

    it("sends a GIF alone, as the server does, and refuses one with a text", async () => {
        const chat = new FakeChatBackend()

        const sent = await chat.sendMessage({countryCode: "fr", text: "", gif})

        expect(sent.gif).toEqual(gif)
        await expect(chat.sendMessage({countryCode: "fr", text: "look", gif}))
            .rejects.toBeInstanceOf(ChatRejectedError)
    })
})
