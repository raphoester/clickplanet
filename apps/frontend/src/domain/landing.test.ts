import {describe, expect, it, vi} from "vitest"
import {firstLanding, Landing, landingOf, recruiterNameOf, withoutLanding} from "./landing.ts"

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

const ADA = "0193f2a4-7c1e-7000-8000-00000000ada0"

function recruiters(held: Record<string, string>) {
    return vi.fn(async (name: string) => held[name.toLowerCase()])
}

describe("landingOf", () => {
    it("reads the tag of a player's link", () => {
        expect(landingOf(page("/play?f=jp&s=rc&r=kutza"), "")).toEqual({tag: "rc", referrer: ""})
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

describe("recruiterNameOf", () => {
    it("reads the name in a player's address, each dash a space", () => {
        expect(recruiterNameOf(page("/play?f=pl&s=rc&r=kutza"))).toBe("kutza")
        expect(recruiterNameOf(page("/play?f=fr&s=rc&r=Ada-Lovelace"))).toBe("Ada Lovelace")
        expect(recruiterNameOf(page(`/play?r=${encodeURIComponent("Émile-Zola")}`))).toBe("Émile Zola")
    })

    it("reads no name from an address with none", () => {
        expect(recruiterNameOf(page("/play?f=pl&s=rc"))).toBeUndefined()
        expect(recruiterNameOf(page("/play?r="))).toBeUndefined()
    })
})

describe("withoutLanding", () => {
    it("takes the tag and the name out of the address and leaves the rest", () => {
        expect(withoutLanding(page("/play?f=fr&s=rc&r=kutza&gfx=halo#top")).href).toBe("https://clickplanet.lol/play?f=fr&gfx=halo#top")
        expect(withoutLanding(page("/play?s=yt")).href).toBe("https://clickplanet.lol/play")
    })
})

describe("firstLanding", () => {
    it("keeps the first page's landing", async () => {
        const kept = store()

        expect(await firstLanding(kept, {tag: "rc", referrer: ""})).toEqual({tag: "rc", referrer: ""})
        expect(kept.write).toHaveBeenCalledWith({tag: "rc", referrer: ""})
    })

    it("keeps it before anything is awaited, so a page that goes at once has kept it", () => {
        const kept = store()

        void firstLanding(kept, {tag: "rc", referrer: "discord.com"}, {name: "kutza", accountOf: () => new Promise(() => {})})

        expect(kept.read()).toEqual({tag: "rc", referrer: "discord.com"})
    })

    it("keeps an empty landing too, so a later link does not take its place", async () => {
        const kept = store()
        await firstLanding(kept, {tag: "", referrer: ""})
        const accountOf = recruiters({kutza: ADA})

        expect(await firstLanding(kept, {tag: "rc", referrer: "discord.com"}, {name: "kutza", accountOf}))
            .toEqual({tag: "", referrer: ""})
        expect(accountOf).not.toHaveBeenCalled()
    })

    it("answers what was kept and writes nothing", async () => {
        const kept = store({tag: "yt", referrer: ""})

        expect(await firstLanding(kept, {tag: "rc", referrer: ""})).toEqual({tag: "yt", referrer: ""})
        expect(kept.write).not.toHaveBeenCalled()
    })

    it("keeps the account that holds the name in the address with the landing", async () => {
        const kept = store()

        const landing = await firstLanding(kept, {tag: "rc", referrer: "www.tiktok.com"}, {name: "Kutza", accountOf: recruiters({kutza: ADA})})

        expect(landing).toEqual({tag: "rc", referrer: "www.tiktok.com", recruiterAccountId: ADA})
        expect(kept.read()).toEqual(landing)
    })

    it("keeps the landing with no recruiter when nobody holds the name, or the name could not be read", async () => {
        const nobody = store()
        expect(await firstLanding(nobody, {tag: "rc", referrer: ""}, {name: "ghost", accountOf: recruiters({})}))
            .toEqual({tag: "rc", referrer: ""})
        expect(nobody.read()).toEqual({tag: "rc", referrer: ""})

        const failed = store()
        const down = vi.fn(async () => {
            throw new Error("the server is down")
        })
        expect(await firstLanding(failed, {tag: "rc", referrer: ""}, {name: "kutza", accountOf: down}))
            .toEqual({tag: "rc", referrer: ""})
    })
})
