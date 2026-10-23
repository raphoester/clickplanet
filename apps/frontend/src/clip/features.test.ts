import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {featuresOf, languageOf, Shown} from "./features.ts"

const SINCE = Date.parse("2026-10-23T09:00:00Z")

const battle: Story = {kind: "battle", attacker: "fr", rival: "pw", victims: ["pw"], place: {country: "fr"}, taken: 900}

const shown: Shown = {
    story: battle, seconds: 12.366666, music: {code: "fr", from: 31.25}, look: "dive",
    since: SINCE, until: SINCE + 2 * 3_600_000 + 15_000,
    tilesChanged: 1_480, storyTilesChanged: 1_210, placeTiles: 6_120, flagTiles: {start: 2_900, end: 3_640},
    landmassesFlipped: 2, bombs: 3, spreads: 41, enclosures: 1, screens: 2.4567, tourStops: 0, solidity: 0.91234,
}

describe("what a clip was", () => {
    it("keeps what the generator chose and what the clip shows, one flat value each in the unit its name says", () => {
        expect(featuresOf(shown).toJson()).toEqual({
            durationSeconds: 12.37, music: "fr", musicStartSeconds: 31.25, camera: "front", look: "dive", storyKind: "battle",
            spanMinutes: 120.3, tilesChanged: 1_480, storyTilesChanged: 1_210, placeTiles: 6_120, flagTilesStart: 2_900,
            flagTilesEnd: 3_640, landmassesFlipped: 2, bombs: 3, spreads: 41, enclosures: 1, screensCrossed: 2.46,
            language: "en", solidity: 0.91,
        })
    })

    it("says a tour is one, with its stops and no look, and a clip with no anthem has none", () => {
        const tour = featuresOf({...shown, look: undefined, music: undefined, tourStops: 3})

        expect([tour.camera, tour.look, tour.tourStops, tour.screensCrossed, tour.music, tour.musicStartSeconds])
            .toEqual(["tour", "", 3, 0, "", 0])
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
