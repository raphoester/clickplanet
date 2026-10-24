import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {Measured, shortlistOf} from "./shortlist.ts"

describe("the shortlist", () => {
    it("gives each story its headline, its key, every country it shows and its teasers, best first", () => {
        const france: Story = {kind: "battle", attacker: "fr", rival: "pw", victims: ["pw"], place: {country: "fr"}, taken: 900}
        const vatican: Story = {kind: "battle", attacker: "va", rival: "pw", victims: ["pw"], place: {country: "va"}, taken: 400}

        const json = shortlistOf([{story: france}, {story: vatican}]).toJson() as {stories: {title: string, key: unknown, countries: string[], teasers?: string[]}[]}

        expect(json.stories.map(({title, key, countries}) => ({title, key, countries}))).toEqual([
            {title: "FRANCE VS PALAU", key: {flag: "fr", continent: "Europe"}, countries: ["fr", "pw"]},
            {title: "VATICAN CITY VS PALAU", key: {flag: "va", continent: "Europe"}, countries: ["va", "pw"]},
        ])
        expect(json.stories[0].teasers?.length).toBeGreaterThan(0)
        expect(json.stories[0].teasers).not.toContain("FRANCE VS PALAU")
    })

    it("keys what a flag holds apart from what it did there", () => {
        const holding: Story = {kind: "holding", attacker: "hr", rival: undefined, victims: ["se"], place: {region: "Europe"}, taken: 9400, share: 0.8}

        expect(shortlistOf([{story: holding}]).toJson()).toMatchObject({
            stories: [{title: "CROATIA HOLDS 80% OF EUROPE", key: {flag: "hr", continent: "Europe", angle: "holding"}, countries: ["hr", "se"]}],
        })
    })

    it("hands the job's editor the facts the generator measured, said as a clip says them", () => {
        const kickout: Story = {kind: "kickout", attacker: "it", rival: undefined, victims: ["gb"], place: {country: "fr"}, taken: 700}
        const measured: Measured = {
            world: false, since: Date.UTC(2026, 9, 24, 10), until: Date.UTC(2026, 9, 24, 12, 30), tilesChanged: 1500,
            storyTilesChanged: 1200, placeTiles: 4000, flagTiles: {start: 3000, end: 3600}, bombs: 2, spreads: 5,
            enclosures: 1, solidity: 0.9134, score: 812.345,
        }

        expect(shortlistOf([{story: kickout, measured}]).toJson()).toMatchObject({
            stories: [{
                title: "ITALY IS KICKING THE UK OUT OF FRANCE",
                facts: {
                    kind: "kickout", place: "France", camera: "front", attacker: "it", takenFrom: ["gb"],
                    names: {it: "Italy", gb: "the UK", fr: "France"}, language: "en", spanMinutes: 150,
                    storyEndedAt: "2026-10-24T12:30:00Z", tilesChanged: 1500, storyTilesChanged: 1200, placeTiles: 4000,
                    flagTilesStart: 3000, flagTilesEnd: 3600, bombs: 2, spreads: 5, enclosures: 1, solidity: 0.91, score: 812.3,
                },
            }],
        })
    })
})
