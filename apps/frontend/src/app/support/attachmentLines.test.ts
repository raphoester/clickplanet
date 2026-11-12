import {describe, expect, it} from "vitest"
import {refusalLine, sizeLine} from "./attachmentLines.ts"

describe("refusalLine", () => {
    it.each([
        [{reason: "too-many", most: 10} as const, "10 files at most."],
        [{reason: "too-many", most: 1} as const, "1 file at most."],
        [{reason: "too-big", name: "clip.mp4", mostBytes: 50 * 1024 * 1024} as const, "clip.mp4 is over 50 MB."],
        [{reason: "not-media", name: "notes.txt"} as const, "notes.txt is not a photo or a video."],
    ])("says %o as a line", (refusal, line) => {
        expect(refusalLine(refusal)).toBe(line)
    })
})

describe("sizeLine", () => {
    it.each([
        [0, "0 B"],
        [900, "900 B"],
        [1024, "1 KB"],
        [120_000, "117 KB"],
        [1024 * 1024, "1 MB"],
        [4_800_000, "4.6 MB"],
        [48_000_000, "46 MB"],
    ])("writes %d bytes as %s", (bytes, line) => {
        expect(sizeLine(bytes)).toBe(line)
    })
})
