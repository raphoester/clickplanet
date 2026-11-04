import {describe, expect, it, vi} from "vitest"
import {fakeAccountId, FAKE_YOU} from "./fakeAccounts.ts"
import {FakePresenceBackend} from "./fakePresenceBackend.ts"
import {NameColor} from "./player.ts"

describe("FakePresenceBackend", () => {
    it("answers a guest's card with its name alone, and no progress, as the server does", async () => {
        const fake = new FakePresenceBackend()
        const guest = fakeAccountId("guest_91aa3d")

        expect(await fake.player(guest)).toEqual({name: "guest_91aa3d", guest: true, admin: false, color: NameColor.UNSPECIFIED})
        expect(await fake.progress(guest)).toBeUndefined()
    })

    it("gives a named player a career, and the title the signed-in player wears", async () => {
        const fake = new FakePresenceBackend()
        const wearable = fake.ownDashboard().wearable
        const chosen = wearable[wearable.length - 1]

        expect(await fake.player(fakeAccountId("Ana"))).toMatchObject({name: "Ana", guest: false, admin: true})
        expect((await fake.progress(fakeAccountId("Ana")))?.streakCurrent).toBe(12)
        expect(fake.wear(chosen.id)).toEqual(chosen)
        expect((await fake.progress(FAKE_YOU))?.wornTitle).toEqual(chosen)
    })

    it("tells a title earned to the pages that listen, and says whether one did", () => {
        const fake = new FakePresenceBackend()
        expect(fake.earnTitle("settler")).toBe(false)

        const onEarned = vi.fn()
        const stop = fake.listenForTitles(onEarned)
        expect(fake.earnTitle("settler")).toBe(true)
        expect(onEarned).toHaveBeenCalledWith(expect.objectContaining({id: "settler"}))

        stop()
        expect(fake.earnTitle("settler")).toBe(false)
    })
})
