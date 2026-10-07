import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {wordsOf} from "./overlay.ts"

const HOUR = 3_600_000

function story(kind: Story["kind"], place: Story["place"], rival?: string): Story {
    return {kind, attacker: "ps", rival, victims: ["fr", "es"], place, taken: 2593}
}

describe("the call at the end of a clip", () => {
    it("shows the flag it asks to defend, which the link opens on", () => {
        const words = wordsOf(story("invasion", {country: "fr"}), 7 * HOUR)

        expect(words.call).toBe("DEFEND FRANCE")
        expect(words.callFlags).toEqual(["fr"])
        expect(words.link).toBe("https://clickplanet.lol/?f=fr")
    })

    it("shows the continent's flag when it asks to defend a continent", () => {
        const words = wordsOf(story("attack", {region: "Europe"}), 2 * HOUR)

        expect(words.call).toBe("DEFEND EUROPE")
        expect(words.callFlags).toEqual(["eu"])
    })

    it("shows no flag for a continent that has none", () => {
        expect(wordsOf(story("attack", {region: "Africa"}), HOUR).callFlags).toEqual([])
    })

    it("shows the flag that strikes back when it asks to fight for it", () => {
        expect(wordsOf(story("comeback", {country: "ps"}), HOUR).callFlags).toEqual(["ps"])
    })

    it("shows both flags when it asks to pick a side", () => {
        expect(wordsOf(story("battle", {country: "fr"}, "fr"), HOUR).callFlags).toEqual(["ps", "fr"])
    })
})
