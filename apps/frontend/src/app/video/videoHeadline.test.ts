import {describe, expect, it} from "vitest"
import {StoryKind, VideoStory} from "../../domain/video/story.ts"
import {videoHeadline} from "./videoHeadline.ts"

const story = (kind: StoryKind, start: number, end: number): VideoStory => ({flag: "pl", country: "de", kind, tiles: 100, start, end})

describe("videoHeadline", () => {
    it("says the flag's day in capitals, as the video does", () => {
        expect(videoHeadline(story("gain", 0, 40))).toBe("POLAND TOOK 40% OF GERMANY")
        expect(videoHeadline(story("loss", 60, 10))).toBe("POLAND IS LOSING GERMANY. HELP")
        expect(videoHeadline(story("quiet", 62, 62))).toBe("POLAND HOLDS 62% OF GERMANY")
    })
})
