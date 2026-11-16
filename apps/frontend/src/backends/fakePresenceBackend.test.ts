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
        const stop = fake.listenForProgression({titleEarned: onEarned, levelReached: vi.fn()})
        expect(fake.earnTitle("settler")).toBe(true)
        expect(onEarned).toHaveBeenCalledWith(expect.objectContaining({id: "settler"}))

        stop()
        expect(fake.earnTitle("settler")).toBe(false)
    })

    it("raises the signed-in player's level once for a jump, as the server does, and draws no level for a guest", async () => {
        const fake = new FakePresenceBackend()
        const levelReached = vi.fn()
        fake.listenForProgression({titleEarned: vi.fn(), levelReached})
        const before = fake.ownLevel()

        expect(fake.giveXp(1).heard).toBe(false)
        const {level, heard} = fake.giveXp(before.nextLevelXp * 3)

        expect(heard).toBe(true)
        expect(level.level).toBeGreaterThan(before.level + 1)
        expect(levelReached).toHaveBeenCalledTimes(1)
        expect(levelReached).toHaveBeenCalledWith(level)
        expect((await fake.progress(FAKE_YOU))?.level).toEqual(level)
        expect((await fake.progress(fakeAccountId("guest_91aa3d")))?.level).toBeUndefined()
    })
})
