import {describe, expect, it} from "vitest"
import {AttachmentRules} from "../backends/support.ts"
import {isPhoto, refusalLine, sizeLine, withPicked} from "./attachments.ts"

const rules: AttachmentRules = {maxPerNote: 3, maxBytes: 1000, contentTypes: ["image/png", "video/mp4"]}

const file = (name: string, type: string, size = 10) => new File([new Uint8Array(size)], name, {type})

describe("withPicked", () => {
    it("adds what was picked after what was chosen, in the order picked", () => {
        const first = file("first.png", "image/png")
        const second = file("second.mp4", "video/mp4")
        const third = file("third.png", "image/png")

        expect(withPicked([first], [second, third], rules)).toEqual({files: [first, second, third], refusal: undefined})
    })

    it("leaves the list it was given as it was", () => {
        const chosen = [file("first.png", "image/png")]

        withPicked(chosen, [file("second.png", "image/png")], rules)

        expect(chosen).toHaveLength(1)
    })

    it("refuses a file that is not a photo or a video, and keeps the others", () => {
        const photo = file("photo.png", "image/png")
        const notes = file("notes.txt", "text/plain")
        const unknown = file("clip.mkv", "")

        expect(withPicked([], [notes, photo, unknown], rules)).toEqual({
            files: [photo], refusal: {reason: "not-media", name: "notes.txt"},
        })
    })

    it("refuses an empty file, which the server would refuse with the whole note", () => {
        const empty = file("empty.png", "image/png", 0)

        expect(withPicked([], [empty], rules)).toEqual({files: [], refusal: {reason: "not-media", name: "empty.png"}})
    })

    it("refuses a file one byte over the most, and takes one at it", () => {
        const atTheMost = file("small.mp4", "video/mp4", 1000)
        const over = file("big.mp4", "video/mp4", 1001)

        expect(withPicked([], [over, atTheMost], rules)).toEqual({
            files: [atTheMost], refusal: {reason: "too-big", name: "big.mp4", mostBytes: 1000},
        })
    })

    it("refuses the files past the most a note carries", () => {
        const four = [1, 2, 3, 4].map((n) => file(`${n}.png`, "image/png"))

        expect(withPicked([], four, rules)).toEqual({files: four.slice(0, 3), refusal: {reason: "too-many", most: 3}})
        expect(withPicked(four.slice(0, 3), [four[3]], rules).files).toHaveLength(3)
    })

    it("takes nothing when the server takes none", () => {
        const none = {maxPerNote: 0, maxBytes: 0, contentTypes: []}

        expect(withPicked([], [file("photo.png", "image/png")], none).files).toEqual([])
    })
})

describe("refusalLine", () => {
    it.each([
        [{reason: "too-many", most: 10} as const, "10 files at most."],
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

describe("isPhoto", () => {
    it("tells a photo from a video", () => {
        expect(isPhoto(file("a.png", "image/png"))).toBe(true)
        expect(isPhoto(file("a.mp4", "video/mp4"))).toBe(false)
    })
})
