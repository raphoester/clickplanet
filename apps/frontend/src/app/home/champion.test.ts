// @vitest-environment jsdom
import {beforeEach, describe, expect, it} from "vitest"
import home from "../../../index.html?raw"
import {sampleEndedSeason} from "../../backends/fakeHallOfFameBackend.ts"
import {EndedSeason} from "../../backends/hallOfFame.ts"
import {showChampion} from "./champion.ts"

const SAMPLE_ENDED_SEASON = sampleEndedSeason()

describe("the champion on the home page", () => {
    let page: Document

    const line = () => page.querySelector<HTMLElement>("[data-champion]")!

    beforeEach(() => {
        page = new DOMParser().parseFromString(home, "text/html")
    })

    it("stays hidden before a season ends", () => {
        showChampion(page, [])

        expect(line().hidden).toBe(true)
    })

    it("names the last season's champion", () => {
        const earlier: EndedSeason = {...SAMPLE_ENDED_SEASON, scores: [{rank: 1, countryCode: "de", points: 90, roundsWon: 3}]}
        showChampion(page, [{...SAMPLE_ENDED_SEASON, number: 1}, earlier])

        expect(line().hidden).toBe(false)
        expect(line().textContent?.replace(/\s+/g, " ").trim()).toBe("Season 1 champion France")
    })

    it("names every country level on first place", () => {
        showChampion(page, [{
            ...SAMPLE_ENDED_SEASON,
            scores: [{rank: 1, countryCode: "de", points: 100, roundsWon: 2}, {rank: 1, countryCode: "fr", points: 100, roundsWon: 2}],
        }])

        expect(page.querySelector("[data-champion-country]")?.textContent).toBe("Germany & France")
    })

    it("stays hidden for a season no country scored in", () => {
        showChampion(page, [{...SAMPLE_ENDED_SEASON, scores: []}])

        expect(line().hidden).toBe(true)
    })
})
