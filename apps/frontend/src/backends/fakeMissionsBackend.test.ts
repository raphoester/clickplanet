import {describe, expect, it, vi} from "vitest"
import {Allegiance} from "../domain/allegiance.ts"
import {reachedOf} from "../domain/missions.ts"
import {AllegianceSource, TileClicker} from "./backend.ts"
import {FakeMissionsBackend} from "./fakeMissionsBackend.ts"

const pledged = (countryId: string): AllegianceSource => ({
    listenForAllegiance: (callback: (allegiance: Allegiance | undefined) => void) => {
        callback({countryId})
        return () => {}
    },
    pledgeAllegiance: async () => {},
    refreshAllegiance: async () => {},
})

const clicker: TileClicker = {clickTile: async () => {}}

describe("FakeMissionsBackend", () => {
    it("puts every mission one tile from done, and the next take finishes and pays them all", async () => {
        const fake = new FakeMissionsBackend(pledged("fr"), () => Date.UTC(2026, 9, 26, 12, 0))
        const clicking = fake.counting(clicker)

        fake.prime()
        const primed = await fake.missions()
        expect(primed.missions.map((mission) => mission.order.goal - reachedOf(mission.order))).toEqual([1, 1, 1, 1, 1])
        expect(primed.missions.some((mission) => mission.paid)).toBe(false)

        await clicking.clickTile(1, "fr")
        const done = await fake.missions()

        expect(done.missions.every((mission) => mission.done && mission.paid)).toBe(true)
    })

    it("answers a rate for every kind of mission it draws", async () => {
        const fake = new FakeMissionsBackend(pledged("fr"), () => Date.UTC(2026, 9, 26, 12, 0))
        const rates = await fake.missionRates()

        for (const mission of (await fake.missions()).missions) expect(rates[mission.order.kind]).toBeDefined()
    })

    it("tells its flag's mission done once, as the server tells the chat", async () => {
        const fake = new FakeMissionsBackend(pledged("fr"), () => Date.UTC(2026, 9, 26, 12, 0))
        const clicking = fake.counting(clicker)
        const onDone = vi.fn()
        fake.listenForFlagMissionsDone(onDone)

        await clicking.clickTile(1, "fr")
        expect(onDone).not.toHaveBeenCalled()

        fake.prime()
        await clicking.clickTile(2, "fr")
        await clicking.clickTile(3, "fr")

        expect(onDone).toHaveBeenCalledTimes(1)
        expect(onDone.mock.lastCall).toEqual(["fr", expect.objectContaining({done: true, points: 2})])
    })

    it("hands its followers the day at once, then again after each take, paid or not", async () => {
        const fake = new FakeMissionsBackend(pledged("fr"), () => Date.UTC(2026, 9, 26, 12, 0))
        const clicking = fake.counting(clicker)
        const onDay = vi.fn()

        fake.follow(onDay)
        await clicking.clickTile(1, "fr")
        expect(onDay).toHaveBeenCalledTimes(2)
        expect(onDay.mock.lastCall?.[0].missions[0].order.taken).toBe(1)

        fake.prime()
        await clicking.clickTile(2, "fr")

        expect(onDay).toHaveBeenCalledTimes(3)
        expect(onDay.mock.lastCall?.[0].missions.every((mission: {paid: boolean}) => mission.paid)).toBe(true)
    })
})
