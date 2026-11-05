// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {Mission} from "../../backends/missions.ts"
import {SEEN_MISSIONS_STORAGE_KEY} from "../../domain/missions.ts"

const bank: Mission = {
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 60},
    reward: {kind: "shields", amount: 10},
    done: true,
    paid: false,
}

afterEach(() => {
    window.localStorage.clear()
    vi.resetModules()
})

const loaded = async () => (await import("./seenMissionsStore.ts")).useSeenMissions

describe("the seen missions", () => {
    it("start from what this browser saved", async () => {
        window.localStorage.setItem(SEEN_MISSIONS_STORAGE_KEY, JSON.stringify({bank: "new"}))

        expect((await loaded()).getState().seen).toEqual({bank: "new"})
    })

    it("save what was shown in this browser", async () => {
        const seen = await loaded()

        seen.getState().markSeen([bank])

        expect(JSON.parse(window.localStorage.getItem(SEEN_MISSIONS_STORAGE_KEY)!)).toEqual({bank: "done"})
    })

    it("tell nobody when what was shown did not change", async () => {
        const seen = await loaded()
        seen.getState().markSeen([bank])
        const told = vi.fn()
        seen.subscribe(told)

        seen.getState().markSeen([bank])

        expect(told).not.toHaveBeenCalled()
    })
})
