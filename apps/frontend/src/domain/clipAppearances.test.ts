import {describe, expect, it} from "vitest"
import {Appearance} from "../backends/clips.ts"
import {TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../links.ts"
import {
    APPEARANCES_READ_GAP_MS,
    appearanceToShow,
    parseShownClips,
    readDue,
    shownAfter,
    watchLinksOf,
} from "./clipAppearances.ts"

function appearance(postId: string, scheduledAt: number, links: Appearance["links"] = [{platform: "tiktok", url: ""}]): Appearance {
    return {postId, title: "FRANCE VS TURKEY", scheduledAt, links}
}

describe("readDue", () => {
    it("reads at once the first time", () => {
        expect(readDue(undefined, 1_000)).toBe(true)
    })

    it("reads again only once the gap has passed", () => {
        expect(readDue(1_000, 1_000 + APPEARANCES_READ_GAP_MS - 1)).toBe(false)
        expect(readDue(1_000, 1_000 + APPEARANCES_READ_GAP_MS)).toBe(true)
    })
})

describe("watchLinksOf", () => {
    it("links each platform to its video", () => {
        expect(watchLinksOf(appearance("post-1", 1, [
            {platform: "tiktok", url: "https://www.tiktok.com/@clickplanetlol/video/1"},
            {platform: "youtube", url: "https://www.youtube.com/shorts/abc"},
        ]))).toEqual([
            {platform: "tiktok", label: "Watch on TikTok", url: "https://www.tiktok.com/@clickplanetlol/video/1"},
            {platform: "youtube", label: "Watch on YouTube", url: "https://www.youtube.com/shorts/abc"},
        ])
    })

    it("links to the channel while the video's address is not known", () => {
        expect(watchLinksOf(appearance("post-1", 1, [{platform: "tiktok", url: ""}, {platform: "youtube", url: ""}])).map((link) => link.url))
            .toEqual([TIKTOK_PROFILE, YOUTUBE_CHANNEL])
    })

    it("gives one link per platform", () => {
        expect(watchLinksOf(appearance("post-1", 1, [
            {platform: "youtube", url: "https://www.youtube.com/shorts/abc"},
            {platform: "youtube", url: ""},
        ]))).toHaveLength(1)
    })
})

describe("appearanceToShow", () => {
    it("shows the newest clip not shown before", () => {
        const older = appearance("post-1", 1_000)
        const newer = appearance("post-2", 2_000)

        expect(appearanceToShow([older, newer], [])).toBe(newer)
        expect(appearanceToShow([newer, older], ["post-2"])).toBe(older)
    })

    it("shows nothing once every clip was shown", () => {
        expect(appearanceToShow([appearance("post-1", 1_000)], ["post-1"])).toBeUndefined()
        expect(appearanceToShow([], [])).toBeUndefined()
    })

    it("skips a clip with nothing to watch", () => {
        expect(appearanceToShow([appearance("post-1", 1_000, [])], [])).toBeUndefined()
    })
})

describe("shownAfter", () => {
    it("keeps every clip of the read, then the ones shown before", () => {
        expect(shownAfter(["post-1"], [appearance("post-3", 3), appearance("post-2", 2), appearance("post-3", 3)]))
            .toEqual(["post-3", "post-2", "post-1"])
    })

    it("keeps a clip once", () => {
        expect(shownAfter(["post-2", "post-1"], [appearance("post-2", 2)])).toEqual(["post-2", "post-1"])
    })

    it("keeps no more than fifty", () => {
        const shown = Array.from({length: 50}, (_, i) => `old-${i}`)

        const kept = shownAfter(shown, [appearance("post-new", 1)])

        expect(kept).toHaveLength(50)
        expect(kept[0]).toBe("post-new")
        expect(kept).not.toContain("old-49")
    })
})

describe("parseShownClips", () => {
    it("reads the ids kept", () => {
        expect(parseShownClips(JSON.stringify(["post-1", "post-2"]))).toEqual(["post-1", "post-2"])
    })

    it("reads nothing kept, or anything else, as none shown", () => {
        expect(parseShownClips(null)).toEqual([])
        expect(parseShownClips("not json")).toEqual([])
        expect(parseShownClips(JSON.stringify({"post-1": true}))).toEqual([])
    })

    it("drops what is not an id", () => {
        expect(parseShownClips(JSON.stringify(["post-1", 7, null]))).toEqual(["post-1"])
    })
})
