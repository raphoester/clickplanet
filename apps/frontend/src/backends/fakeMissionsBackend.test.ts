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
})

const clicker: TileClicker = {clickTile: async () => {}}

describe("FakeMissionsBackend", () => {
    it("puts every mission one tile from done, and the next take finishes and pays them all", async () => {
        const grant = vi.fn()
        const fake = new FakeMissionsBackend(grant, pledged("fr"), () => Date.UTC(2026, 9, 26, 12, 0))
        const clicking = fake.counting(clicker)

        fake.prime()
        const primed = await fake.missions()
        expect(primed.missions.map((mission) => mission.order.goal - reachedOf(mission.order))).toEqual([1, 1, 1, 1, 1])
        expect(grant).not.toHaveBeenCalled()

        await clicking.clickTile(1, "fr")
        const done = await fake.missions()

        expect(done.missions.every((mission) => mission.done && mission.paid)).toBe(true)
        expect(grant.mock.calls.map(([kind]) => kind)).toEqual(["shields", "spreadClicks", "bomb", "encloseClicks", "shields"])
    })
})
