import {describe, expect, it} from "vitest"
import {ANNOUNCE_EVERY_MS, Announcing, PresenceSchedule, SETTLE_MS} from "./presence.ts"

const france: Announcing = {countryCode: "fr", guestName: "Bo"}

/** Claims and settles at once, as an announce that landed. */
function sent(schedule: PresenceSchedule, now: number, session: string | undefined) {
    const presence = schedule.claim(now, session)
    schedule.settle()
    return presence
}

describe("PresenceSchedule", () => {
    // A mint is a Turnstile check: a visitor who never clicked is not listed.
    it("announces nothing while no token is held", () => {
        const schedule = new PresenceSchedule(france)

        expect(sent(schedule, 0, undefined)).toBeUndefined()
        expect(sent(schedule, 10 * ANNOUNCE_EVERY_MS, undefined)).toBeUndefined()
    })

    it("announces as soon as a token is held, then not again until it is due", () => {
        const schedule = new PresenceSchedule(france)

        expect(sent(schedule, 1_000, "token-1")).toEqual({countryCode: "fr", guestName: "Bo"})
        expect(sent(schedule, 2_000, "token-1")).toBeUndefined()
        expect(sent(schedule, 1_000 + ANNOUNCE_EVERY_MS - 1, "token-1")).toBeUndefined()
    })

    it("announces again every 30s", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        expect(sent(schedule, ANNOUNCE_EVERY_MS, "token-1")).toBeDefined()
        expect(sent(schedule, 2 * ANNOUNCE_EVERY_MS - 1, "token-1")).toBeUndefined()
        expect(sent(schedule, 2 * ANNOUNCE_EVERY_MS, "token-1")).toBeDefined()
    })

    // A sign-in invalidates the token, and the next one names another account.
    it("announces at once under a new token", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        expect(sent(schedule, 5_000, "token-2")).toBeDefined()
    })

    it("stops while the token is dropped, and announces when a click mints the next", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        expect(sent(schedule, ANNOUNCE_EVERY_MS, undefined)).toBeUndefined()
        expect(sent(schedule, ANNOUNCE_EVERY_MS + 5_000, "token-2")).toBeDefined()
    })

    it("announces a new flag once it has held still for a second", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        schedule.want({...france, countryCode: "jp"}, 5_000)

        expect(sent(schedule, 5_000 + SETTLE_MS - 1, "token-1")).toBeUndefined()
        expect(sent(schedule, 5_000 + SETTLE_MS, "token-1")).toEqual({countryCode: "jp", guestName: "Bo"})
    })

    it("waits for the last of several quick changes", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        schedule.want({...france, countryCode: "jp"}, 5_000)
        schedule.want({...france, countryCode: "de"}, 5_800)

        expect(sent(schedule, 6_000, "token-1")).toBeUndefined()
        expect(sent(schedule, 5_800 + SETTLE_MS, "token-1")).toMatchObject({countryCode: "de"})
    })

    it("announces a new guest name, and a new username, the same way", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        schedule.want({...france, guestName: "Yuki"}, 5_000)
        expect(sent(schedule, 5_000 + SETTLE_MS, "token-1")).toMatchObject({guestName: "Yuki"})

        schedule.want({...france, guestName: "Yuki", username: "yuki_jp"}, 9_000)
        expect(sent(schedule, 9_000 + SETTLE_MS, "token-1")).toBeDefined()
    })

    it("does not announce a change that was undone before it settled", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        schedule.want({...france, countryCode: "jp"}, 5_000)
        schedule.want(france, 5_500)

        expect(sent(schedule, 10_000, "token-1")).toBeUndefined()
    })

    it("does not count a change made before the first announce against it", () => {
        const schedule = new PresenceSchedule(france)
        schedule.want({...france, countryCode: "jp"}, 0)

        expect(sent(schedule, 100, "token-1")).toMatchObject({countryCode: "jp"})
    })

    it("sends one announce at a time", () => {
        const schedule = new PresenceSchedule(france)

        expect(schedule.claim(0, "token-1")).toBeDefined()
        schedule.want({...france, countryCode: "jp"}, 100)
        expect(schedule.claim(100 + SETTLE_MS, "token-1")).toBeUndefined()

        schedule.settle()
        expect(schedule.claim(100 + SETTLE_MS, "token-1")).toMatchObject({countryCode: "jp"})
    })

    // A failure is not retried on the next tick: that would hammer a server that is down.
    it("waits the full interval after an announce that failed", () => {
        const schedule = new PresenceSchedule(france)
        sent(schedule, 0, "token-1")

        expect(sent(schedule, 1_000, "token-1")).toBeUndefined()
        expect(sent(schedule, ANNOUNCE_EVERY_MS, "token-1")).toBeDefined()
    })
})
