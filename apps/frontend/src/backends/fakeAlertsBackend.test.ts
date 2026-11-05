import {describe, expect, it} from "vitest"
import {FakeAlertsBackend} from "./fakeAlertsBackend.ts"

describe("FakeAlertsBackend", () => {
    it("links a signed-in account once the join has landed, as the backend does a moment after the sign-in", async () => {
        let now = 10_000
        let signedIn = false
        const fake = new FakeAlertsBackend(async () => signedIn, 1_500, () => now)

        expect(await fake.discordLink()).toMatchObject({offered: true, linked: false})
        signedIn = true
        expect((await fake.discordLink()).linked).toBe(false)
        now += 1_499
        expect((await fake.discordLink()).linked).toBe(false)
        now += 1
        expect((await fake.discordLink()).linked).toBe(true)
    })

    it("forgets the join when the account signs out", async () => {
        let now = 10_000
        let signedIn = true
        const fake = new FakeAlertsBackend(async () => signedIn, 1_500, () => now)
        await fake.discordLink()
        now += 1_500

        signedIn = false
        expect((await fake.discordLink()).linked).toBe(false)
        signedIn = true
        expect((await fake.discordLink()).linked).toBe(false)
    })
})
