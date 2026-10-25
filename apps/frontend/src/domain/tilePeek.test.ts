import {describe, expect, it} from "vitest"
import {agoOf, holderShown, TilePeek} from "./tilePeek.ts"

const RULES = {restSeconds: 0.5, holdSeconds: 0.5, tolerancePx: 12}

const at = (x: number, y = 100) => ({x, y})

const finger = (clientX = 100, clientY = 100) => ({pointerId: 1, clientX, clientY, isPrimary: true})

describe("TilePeek", () => {
    it("peeks at a tile the mouse rests on, once, where the mouse is now", () => {
        const peek = new TilePeek(RULES)
        peek.hover(7, 10, at(100))
        peek.hover(7, 10.2, at(104))

        expect(peek.tick(10.4)).toBeUndefined()
        expect(peek.tick(10.5)).toEqual({tile: 7, point: at(104), touch: false})
        expect(peek.tick(11)).toBeUndefined()
        expect(peek.peeked).toBe(false)
    })

    it("starts the rest again on each new tile, and not on the same one", () => {
        const peek = new TilePeek(RULES)
        peek.hover(7, 10, at(100))
        peek.hover(8, 10.3, at(110))
        peek.hover(8, 10.6, at(112))

        expect(peek.tick(10.7)).toBeUndefined()
        expect(peek.tick(10.8)?.tile).toBe(8)
    })

    it("does not peek at a tile just clicked, nor once the mouse left", () => {
        const peek = new TilePeek(RULES)
        peek.hover(7, 10, at(100))
        peek.clicked()
        expect(peek.tick(11)).toBeUndefined()

        peek.hover(8, 11, at(110))
        peek.hover(undefined, 11.2, at(110))
        expect(peek.tick(12)).toBeUndefined()
    })

    it("peeks at a tile a finger holds still, where the finger went down, and swallows the click that ends the press", () => {
        const peek = new TilePeek(RULES)
        peek.begin(finger(), 7, 10)
        peek.move({pointerId: 1, clientX: 108, clientY: 100})

        expect(peek.tick(10.5)).toEqual({tile: 7, point: at(100), touch: true})
        expect(peek.peeked).toBe(true)

        peek.begin(finger(), 9, 20)
        expect(peek.peeked).toBe(false)
    })

    it("lets a tap or a drag be what it is", () => {
        const peek = new TilePeek(RULES)
        peek.begin(finger(), 7, 10)
        peek.end(1)
        expect(peek.tick(11)).toBeUndefined()

        peek.begin(finger(), 7, 20)
        peek.move({pointerId: 1, clientX: 120, clientY: 100})
        expect(peek.tick(21)).toBeUndefined()
        expect(peek.peeked).toBe(false)
    })

    it("lets a second finger cancel the press, for a pinch", () => {
        const peek = new TilePeek(RULES)
        peek.begin(finger(), 7, 10)
        peek.begin({...finger(), pointerId: 2, isPrimary: false}, 8, 10.1)

        expect(peek.tick(11)).toBeUndefined()
    })
})

describe("holderShown", () => {
    const peek = {tile: 7, countryCode: "fr", point: at(100), touch: false}
    const holder = {accountId: "account-ana", countryCode: "fr", takenAt: 0}

    it("is the holder of a tile that still wears its flag", () => {
        expect(holderShown(holder, peek, "account-me")).toBe(holder)
    })

    it("is nobody for a tile that changed hands since, or that the player holds itself", () => {
        expect(holderShown(holder, {...peek, countryCode: "de"}, "account-me")).toBeUndefined()
        expect(holderShown(holder, peek, "account-ana")).toBeUndefined()
        expect(holderShown(undefined, peek, "account-me")).toBeUndefined()
    })
})

describe("agoOf", () => {
    const now = 1_761_393_600_000

    it("says how long ago, in the largest whole unit", () => {
        expect(agoOf(now - 20_000, now)).toBe("just now")
        expect(agoOf(now - 3 * 60_000 - 59_000, now)).toBe("3 min ago")
        expect(agoOf(now - 60 * 60_000, now)).toBe("1 hour ago")
        expect(agoOf(now - 5 * 60 * 60_000, now)).toBe("5 hours ago")
        expect(agoOf(now - 24 * 60 * 60_000, now)).toBe("1 day ago")
        expect(agoOf(now - 40 * 24 * 60 * 60_000, now)).toBe("40 days ago")
    })

    it("reads a take stamped ahead of this clock as just now", () => {
        expect(agoOf(now + 5_000, now)).toBe("just now")
    })
})
