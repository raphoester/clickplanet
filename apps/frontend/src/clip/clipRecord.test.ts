import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {clipRecordOf, languageOf, Shown} from "./clipRecord.ts"

const SINCE = Date.parse("2026-10-23T09:00:00Z")

const battle: Story = {kind: "battle", attacker: "fr", rival: "pw", victims: ["pw"], place: {country: "fr"}, taken: 900}

const shown: Shown = {
    story: battle, generator: "8dc31d2fb470e667a4853de7674ed209d3efc945", seconds: 12.366666, music: {code: "fr", from: 31.25},
    look: "dive", since: SINCE, until: SINCE + 2 * 3_600_000 + 15_000,
    tilesChanged: 1_480, storyTilesChanged: 1_210, placeTiles: 6_120, flagTiles: {start: 2_900, end: 3_640},
    landmassesFlipped: 2, bombs: 3, spreads: 41, enclosures: 1, screens: 2.4567, tourStops: 0, solidity: 0.91234, variants: {},
}

describe("what a clip was", () => {
    it("keeps what the generator chose and what the clip shows, under its format, each code a word", () => {
        expect(clipRecordOf(shown).toJson()).toEqual({
            format: "globe-replay",
            generator: "8dc31d2fb470e667a4853de7674ed209d3efc945",
            common: {durationSeconds: 12.37, music: "fr", musicStartSeconds: 31.25, language: "en"},
            globe: {
                camera: "front", look: "dive", storyKind: "battle", spanMinutes: 120.3, tilesChanged: 1_480,
                storyTilesChanged: 1_210, placeTiles: 6_120, flagTilesStart: 2_900, flagTilesEnd: 3_640, landmassesFlipped: 2,
                bombs: 3, spreads: 41, enclosures: 1, screensCrossed: 2.46, solidity: 0.91,
            },
        })
    })

    it("says a tour is one, with its stops, and says none for the look and the anthem it has none of", () => {
        const tour = clipRecordOf({...shown, look: undefined, music: undefined, tourStops: 3})

        expect([tour.globe?.camera, tour.globe?.look, tour.globe?.tourStops, tour.globe?.screensCrossed])
            .toEqual(["tour", "none", 3, 0])
        expect([tour.common?.music, tour.common?.musicStartSeconds]).toEqual(["none", 0])
    })

    it("keeps the arm of each experiment the clip was made in, and none when it is in none", () => {
        expect(clipRecordOf({...shown, variants: {length: "short"}}).toJson()).toMatchObject({variants: {length: "short"}})
        expect(clipRecordOf(shown).toJson()).not.toHaveProperty("variants")
    })

    it("names an unknown commit", () => {
        expect(clipRecordOf({...shown, generator: ""}).generator).toBe("unknown")
    })

    it("hears a call-up in its country's language, and every other clip in English", () => {
        const callUp: Story = {
            kind: "callup", attacker: "pl", rival: undefined, victims: ["dz"], place: {region: "the world"}, taken: 275,
            finaleAt: Date.parse("2026-10-31T21:00:00Z"),
        }

        expect(languageOf(callUp)).toBe("pl")
        expect(languageOf({...callUp, attacker: "gb"})).toBe("en")
        expect(languageOf(battle)).toBe("en")
    })
})
