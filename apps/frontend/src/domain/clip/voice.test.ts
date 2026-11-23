import {describe, expect, it} from "vitest"
import {Said, subtitleAt, subtitlesOf, voicedTimelineOf, VOICE_AT_SECONDS} from "./voice.ts"

const texts = (lines: Said[]) => subtitlesOf(lines).map(({text}) => text)

describe("the subtitles' pieces", () => {
    it("are a line of 6 words or fewer, whole", () => {
        expect(texts([{text: "And Sweden just keeps creeping up.", from: 0, to: 2}])).toEqual(["And Sweden just keeps creeping up."])
    })

    it("are cut at each comma, semicolon and colon first", () => {
        expect(texts([{text: "Okay, so it's day twenty three, and Bulgaria rules India.", from: 0, to: 4}]))
            .toEqual(["Okay,", "so it's day twenty three,", "and Bulgaria rules India."])
        expect(texts([{text: "One thing; then another: and a third", from: 0, to: 3}])).toEqual(["One thing;", "then another:", "and a third"])
    })

    it("are cut evenly where a phrase is longer than 6 words, the longer first", () => {
        expect(texts([{text: "France has been stuck at about a sixth for almost four days.", from: 0, to: 3}]))
            .toEqual(["France has been stuck at about", "a sixth for almost four days."])
        expect(texts([{text: "one two three four five six seven", from: 0, to: 2}])).toEqual(["one two three four", "five six seven"])
        expect(texts([{text: "a b c d e f g h i j k l m", from: 0, to: 2}])).toEqual(["a b c d e", "f g h i", "j k l m"])
    })

    it("run on across the lines, in the order they are said", () => {
        expect(texts([{text: "Yeah, Bulgaria.", from: 0, to: 1}, {text: "But Turkey keeps pushing in.", from: 1.33, to: 3}]))
            .toEqual(["Yeah,", "Bulgaria.", "But Turkey keeps pushing in."])
    })
})

describe("the subtitles' times", () => {
    it("start when the voice does, 0.4 s into the clip", () => {
        const [first] = subtitlesOf([{text: "Nope.", from: 0, to: 0.5}])

        expect(VOICE_AT_SECONDS).toBe(0.4)
        expect(first.from).toBeCloseTo(0.4)
    })

    it("share a line's time by each piece's characters", () => {
        const [okay, rest] = subtitlesOf([{text: "Okay, so it's day twenty-three.", from: 1, to: 4.1}])

        expect(okay.from).toBeCloseTo(1.4)
        expect(rest.from).toBeCloseTo(1.4 + 3.1 * 5 / 30)
    })

    it("show each piece until the next starts, over the breath between two lines too", () => {
        const [yeah, bulgaria, turkey] = subtitlesOf([{text: "Yeah, Bulgaria.", from: 0, to: 1}, {text: "But Turkey keeps pushing in.", from: 1.33, to: 3}])

        expect(yeah.until).toBeCloseTo(bulgaria.from)
        expect(bulgaria.until).toBeCloseTo(turkey.from)
        expect(turkey.from).toBeCloseTo(1.73)
    })

    it("hold the last piece 0.4 s past its line's end", () => {
        expect(subtitlesOf([{text: "Nope.", from: 0, to: 0.5}])[0].until).toBeCloseTo(1.3)
    })

    it("show one piece at a time, and none before the voice or after the last", () => {
        const subtitles = subtitlesOf([{text: "Yeah, Bulgaria.", from: 0, to: 1}])

        expect(subtitleAt(subtitles, 0.2)).toBeUndefined()
        expect(subtitleAt(subtitles, 0.5)?.text).toBe("Yeah,")
        expect(subtitleAt(subtitles, 1.3)?.text).toBe("Bulgaria.")
        expect(subtitleAt(subtitles, 1.8)).toBeUndefined()
    })
})

describe("the length of a clip with a voice", () => {
    it("is the voice from 0.4 s in, its last words read 0.4 s more, then the 2.5 s call", () => {
        const lines = [{text: "Nope.", from: 0, to: 0.5}, {text: "So what's France waiting for?", from: 15, to: 17.3}]
        const timeline = voicedTimelineOf(lines)

        expect(timeline).toEqual({seconds: 20.6, ending: 2.5})
        expect(subtitlesOf(lines).at(-1)?.until).toBeCloseTo(timeline.seconds - timeline.ending)
    })

    it("passes 22 s rather than cut the voice", () => {
        expect(voicedTimelineOf([{text: "A long one.", from: 0, to: 19.2}]).seconds).toBe(22.5)
    })

    it("keeps the call it is given, a day's podium", () => {
        expect(voicedTimelineOf([{text: "France takes day five.", from: 0, to: 10}], 4)).toEqual({seconds: 14.8, ending: 4})
    })

    it("refuses a voice past what the render job has time for, and no voice at all", () => {
        expect(() => voicedTimelineOf([{text: "Far too long.", from: 0, to: 28}])).toThrow("past the 30 s the render job has time for")
        expect(() => voicedTimelineOf([])).toThrow("it has no line")
    })
})
