import {describe, expect, it} from "vitest"
import {
    ASK_SPACING_MS,
    askClosed,
    askDue,
    askedOf,
    askOpened,
    AskScene,
    NOTHING_ASKED,
    SALE_LAST_CALL_MS,
    SALE_NUDGE_EVERY_MS,
    saleKeyOf,
    saleNudgeDue,
    saleSeenOf,
} from "./nudges.ts"
import type {Sale} from "./shop.ts"

const HOUR = 60 * 60_000
const DAY = 24 * HOUR
const now = Date.UTC(2026, 10, 12, 18)
const scene: AskScene = {guided: false, battlePlays: false, saleRuns: false}

describe("askDue", () => {
    it("asks for a coffee first, then takes turns with the follows", () => {
        expect(askDue(NOTHING_ASKED, null, scene, now)).toBe("coffee")

        const coffee = askClosed(NOTHING_ASKED, "coffee", now)
        expect(askDue(coffee, null, scene, now + ASK_SPACING_MS)).toBe("tiktok")

        const tiktok = askClosed(coffee, "tiktok", now + ASK_SPACING_MS)
        expect(askDue(tiktok, null, scene, now + 2 * ASK_SPACING_MS)).toBe("youtube")

        const youtube = askClosed(tiktok, "youtube", now + 2 * ASK_SPACING_MS)
        expect(askDue(youtube, null, scene, now + 3 * ASK_SPACING_MS)).toBe("coffee")
    })

    it("waits its hours after the last ask, and after the battle's teaser", () => {
        const coffee = askClosed(NOTHING_ASKED, "coffee", now)
        expect(askDue(coffee, null, scene, now + ASK_SPACING_MS - 1)).toBeUndefined()

        const teased = {key: "tonight", at: now}
        expect(askDue(NOTHING_ASKED, teased, scene, now + ASK_SPACING_MS - 1)).toBeUndefined()
        expect(askDue(NOTHING_ASKED, teased, scene, now + ASK_SPACING_MS)).toBe("coffee")
    })

    it("asks nothing on a first visit or in a battle", () => {
        expect(askDue(NOTHING_ASKED, null, {...scene, guided: true}, now)).toBeUndefined()
        expect(askDue(NOTHING_ASKED, null, {...scene, battlePlays: true}, now)).toBeUndefined()
    })

    it("asks nothing while a sale runs: its nudge takes the asks' turn", () => {
        expect(askDue(NOTHING_ASKED, null, {...scene, saleRuns: true}, now)).toBeUndefined()
    })

    it("never asks for a follow again once its link was opened", () => {
        const followed = askOpened(askOpened(NOTHING_ASKED, "tiktok", now - 2 * DAY), "youtube", now - DAY)
        expect(askDue(followed, null, scene, now)).toBe("coffee")

        const all = askOpened(followed, "coffee", now)
        expect(askDue(all, null, scene, now + 29 * DAY)).toBeUndefined()
        expect(askDue(all, null, scene, now + 30 * DAY)).toBe("coffee")
    })
})

describe("askedOf", () => {
    it("reads what was kept, and nothing from anything else", () => {
        const asked = askOpened(askClosed(NOTHING_ASKED, "coffee", now), "tiktok", now + HOUR)
        expect(askedOf(JSON.stringify(asked))).toEqual(asked)
        expect(askedOf(null)).toEqual(NOTHING_ASKED)
        expect(askedOf("coffee")).toEqual(NOTHING_ASKED)
        expect(askedOf("null")).toEqual(NOTHING_ASKED)
        expect(askedOf(JSON.stringify({shown: {coffee: "today", snapchat: now}, opened: 3}))).toEqual(NOTHING_ASKED)
    })
})

describe("saleNudgeDue", () => {
    const sale: Sale = {percentOff: 50, endsAt: now + 3 * DAY}
    const seenAt = (at: number) => ({key: saleKeyOf(sale), at})

    it("shows the sale on the first visit while it runs, and no sale that ended", () => {
        expect(saleNudgeDue(sale, null, scene, now)).toBe(true)
        expect(saleNudgeDue(sale, {key: "an older sale", at: now - HOUR}, scene, now)).toBe(true)
        expect(saleNudgeDue(sale, null, scene, sale.endsAt)).toBe(false)
        expect(saleNudgeDue(undefined, null, scene, now)).toBe(false)
    })

    it("shows it again once a day at most", () => {
        expect(saleNudgeDue(sale, seenAt(now), scene, now + SALE_NUDGE_EVERY_MS - 1)).toBe(false)
        expect(saleNudgeDue(sale, seenAt(now), scene, now + SALE_NUDGE_EVERY_MS)).toBe(true)
    })

    it("shows it once more in its last hours, once", () => {
        const lastCall = sale.endsAt - SALE_LAST_CALL_MS
        expect(saleNudgeDue(sale, seenAt(lastCall - HOUR), scene, lastCall - 1)).toBe(false)
        expect(saleNudgeDue(sale, seenAt(lastCall - HOUR), scene, lastCall)).toBe(true)
        expect(saleNudgeDue(sale, seenAt(lastCall), scene, lastCall + HOUR)).toBe(false)
    })

    it("shows nothing on a first visit or in a battle", () => {
        expect(saleNudgeDue(sale, null, {...scene, guided: true}, now)).toBe(false)
        expect(saleNudgeDue(sale, null, {...scene, battlePlays: true}, now)).toBe(false)
    })

    it("reads what the browser kept, and nothing of what it cannot read", () => {
        expect(saleSeenOf(JSON.stringify(seenAt(now)))).toEqual(seenAt(now))
        expect(saleSeenOf(null)).toBeNull()
        expect(saleSeenOf("{")).toBeNull()
        expect(saleSeenOf(JSON.stringify({key: 3, at: now}))).toBeNull()
    })
})
