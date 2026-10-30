import {describe, expect, it, vi} from "vitest"
import {firstLanding, Landing, landingOf, withoutLanding} from "./landing.ts"
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
    it("reads the tag of a player's link", () => {
        expect(landingOf(new URL(shareUrl("jp")), "")).toEqual({tag: "rc", referrer: ""})
    })

    it("reads the tag of the YouTube link", () => {
        expect(landingOf(page("/play?s=YT"), "")).toEqual({tag: "yt", referrer: ""})
    })

    it("keeps only the host of the page before", () => {
        expect(landingOf(page("/"), "https://www.Google.com/search?q=clickplanet")).toEqual({tag: "", referrer: "www.google.com"})
    })

    it("keeps no referrer for a page of the game's own, or none at all", () => {
        expect(landingOf(page("/play"), "https://clickplanet.lol/?f=fr").referrer).toBe("")
        expect(landingOf(page("/play"), "").referrer).toBe("")
    })
})

describe("withoutLanding", () => {
    it("takes the tag out of the address and leaves the rest", () => {
        expect(withoutLanding(page("/play?f=fr&s=rc&gfx=halo#top")).href).toBe("https://clickplanet.lol/play?f=fr&gfx=halo#top")
        expect(withoutLanding(page("/play?s=yt")).href).toBe("https://clickplanet.lol/play")
    })
})

describe("firstLanding", () => {
    it("keeps the first page's landing", () => {
        const kept = store()

        expect(firstLanding(kept, {tag: "rc", referrer: ""})).toEqual({tag: "rc", referrer: ""})
        expect(kept.write).toHaveBeenCalledWith({tag: "rc", referrer: ""})
    })

    it("keeps an empty landing too, so a later link does not take its place", () => {
        const kept = store()
        firstLanding(kept, {tag: "", referrer: ""})

        expect(firstLanding(kept, {tag: "rc", referrer: "discord.com"})).toEqual({tag: "", referrer: ""})
    })

    it("answers what was kept and writes nothing", () => {
        const kept = store({tag: "yt", referrer: ""})

        expect(firstLanding(kept, {tag: "rc", referrer: ""})).toEqual({tag: "yt", referrer: ""})
        expect(kept.write).not.toHaveBeenCalled()
    })
})
