import {describe, expect, it, vi} from "vitest"
import {firstLanding, hasLanding, Landing, landingOf, withoutLanding} from "./landing.ts"
import {shareUrl} from "./shareCard.ts"

function page(path: string): URL {
    return new URL(path, "https://clickplanet.lol")
}

function store(kept?: Landing) {
    let held = kept
    return {
        read: vi.fn(() => held),
        write: vi.fn((landing: Landing) => {
            held = landing
        }),
    }
}

describe("landingOf", () => {
    it("reads the flag of a recruit link", () => {
        expect(landingOf(new URL(shareUrl("jp")))).toEqual({flag: "jp", tag: ""})
    })

    it("reads the tag of the YouTube link", () => {
        expect(landingOf(page("/play?s=YT"))).toEqual({flag: "", tag: "yt"})
    })

    it("drops a flag that is no country", () => {
        expect(landingOf(page("/play?f=zz&s=yt"))).toEqual({flag: "", tag: "yt"})
    })

    it("is empty on a page with no query", () => {
        expect(landingOf(page("/play"))).toEqual({flag: "", tag: ""})
    })
})

describe("withoutLanding", () => {
    it("takes the flag and the tag out of the address and leaves the rest", () => {
        expect(withoutLanding(page("/play?f=fr&s=yt&gfx=halo#top")).href).toBe("https://clickplanet.lol/play?gfx=halo#top")
        expect(withoutLanding(page("/play?f=fr")).href).toBe("https://clickplanet.lol/play")
    })

    it("is asked for only when there is something to take out", () => {
        expect(hasLanding(page("/play?s=yt"))).toBe(true)
        expect(hasLanding(page("/play?f=fr"))).toBe(true)
        expect(hasLanding(page("/play?gfx=halo"))).toBe(false)
    })
})

describe("firstLanding", () => {
    it("keeps the first page's landing", () => {
        const kept = store()

        expect(firstLanding(kept, {flag: "fr", tag: ""})).toEqual({flag: "fr", tag: ""})
        expect(kept.write).toHaveBeenCalledWith({flag: "fr", tag: ""})
    })

    it("keeps an empty landing too, so a later link does not take its place", () => {
        const kept = store()
        firstLanding(kept, {flag: "", tag: ""})

        expect(firstLanding(kept, {flag: "de", tag: ""})).toEqual({flag: "", tag: ""})
    })

    it("answers what was kept and writes nothing", () => {
        const kept = store({flag: "", tag: "yt"})

        expect(firstLanding(kept, {flag: "it", tag: ""})).toEqual({flag: "", tag: "yt"})
        expect(kept.write).not.toHaveBeenCalled()
    })
})
