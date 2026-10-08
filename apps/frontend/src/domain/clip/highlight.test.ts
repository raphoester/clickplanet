import {describe, expect, it} from "vitest"
import {highlightsOf, Sound, soundsOf} from "./highlight.ts"

const RATE = 22050
const HOP = 0.1
const TUNE = 0.02
const DRUMS = 0.3

function struck(seconds: number, loud: number, flat: number): Omit<Sound, "at">[] {
    return Array.from({length: Math.round(seconds / HOP)}, (_, i) => ({loud: loud - i % 5, flat}))
}

function rest(seconds: number): Omit<Sound, "at">[] {
    return Array.from({length: Math.round(seconds / HOP)}, () => ({loud: 0, flat: TUNE}))
}

function played(...parts: Omit<Sound, "at">[][]): Sound[] {
    return parts.flat().map((sound, i) => ({...sound, at: i * HOP}))
}

function tone(seconds: number, gain: number): Float32Array {
    return Float32Array.from({length: seconds * RATE}, (_, i) =>
        gain * [1, 2, 3, 4].reduce((sum, harmonic) => sum + Math.sin(2 * Math.PI * 330 * harmonic * i / RATE) / harmonic, 0))
}

function noise(seconds: number): Float32Array {
    let seed = 1
    return Float32Array.from({length: seconds * RATE}, () => {
        seed = (seed * 1103515245 + 12345) % 2 ** 31
        return seed / 2 ** 30 - 1
    })
}

describe("the sound of an anthem", () => {
    it("tells a drum roll from a tune", () => {
        const drums = soundsOf(noise(2), RATE)
        const tune = soundsOf(tone(2, 0.3), RATE)

        expect(Math.min(...drums.map((sound) => sound.flat))).toBeGreaterThan(DRUMS)
        expect(Math.max(...tune.map((sound) => sound.flat))).toBeLessThan(TUNE)
    })

    it("hears the louder band louder", () => {
        const soft = soundsOf(tone(1, 0.1), RATE)[0].loud
        const loud = soundsOf(tone(1, 0.3), RATE)[0].loud

        expect(loud - soft).toBeCloseTo(20 * Math.log10(3), 1)
    })
})

describe("the highlights of an anthem", () => {
    const anthem = played(struck(10, 20, DRUMS), struck(19.5, 18, TUNE), rest(0.5), struck(10, 24, TUNE), struck(20, 18, TUNE))

    it("start with the loudest phrase after a pause", () => {
        expect(highlightsOf(anthem)[0]).toBeCloseTo(30)
    })

    it("never start on the drum roll before the band plays", () => {
        expect(Math.min(...highlightsOf(anthem))).toBeGreaterThanOrEqual(10)
    })

    it("are moments apart", () => {
        const highlights = highlightsOf(anthem)

        expect(highlights.length).toBeGreaterThan(1)
        for (const [i, at] of highlights.entries()) {
            for (const other of highlights.slice(i + 1)) expect(Math.abs(at - other)).toBeGreaterThanOrEqual(8)
        }
    })

    it("end with the band's first notes, so a long clip still starts on its tune", () => {
        const highlights = highlightsOf(played(struck(10, 20, DRUMS), struck(10, 18, TUNE), rest(0.5), struck(30, 24, TUNE)))

        expect(highlights[highlights.length - 1]).toBeLessThan(11)
    })

    it("are none when the band never plays its tune", () => {
        expect(highlightsOf(played(struck(30, 20, DRUMS)))).toEqual([])
    })
})
