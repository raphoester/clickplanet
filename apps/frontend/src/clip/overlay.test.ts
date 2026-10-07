import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {wordsOf} from "./overlay.ts"

function story(kind: Story["kind"], place: Story["place"], rival?: string): Story {
    return {kind, attacker: "ps", rival, victims: ["fr", "es"], place, taken: 2593}
}

describe("the headline of a clip", () => {
    it("says who kicks whom out when the attacker already held most of the place", () => {
        const words = wordsOf({...story("kickout", {country: "au"}), attacker: "fr", victims: ["pl"]})

        expect(words.headline).toBe("FRANCE IS KICKING POLAND OUT OF AUSTRALIA")
        expect(words.call).toBe("PICK A SIDE")
        expect(words.callFlags).toEqual(["fr", "pl"])
    })
})

describe("the line under the headline", () => {
    it("names the battle and nothing else", () => {
        expect(wordsOf(story("battle", {country: "fr"}, "nl")).line).toBe("The battle for France")
    })

    it("is left out when it would only count tiles and hours", () => {
        expect(wordsOf(story("invasion", {country: "tr"})).line).toBeUndefined()
        expect(wordsOf(story("attack", {region: "Europe"})).line).toBeUndefined()
        expect(wordsOf(story("comeback", {country: "ps"})).line).toBeUndefined()
    })
})

describe("the caption", () => {
    it("asks a question, names the site as text, and tags the place but not the attacker", () => {
        expect(wordsOf(story("invasion", {country: "tr"})).caption).toBe(
            "PALESTINE IS INVADING TURKEY. Who stops them? 👇\nclickplanet.lol\n#clickplanet #pixelwars #rplace #wplace #map #turkey")
    })

    it("asks to pick a side in a battle", () => {
        expect(wordsOf(story("battle", {country: "fr"}, "nl")).caption).toMatch(/^PALESTINE VS NETHERLANDS\. Pick a side 👇\n/)
    })

    it("tags no place for the world", () => {
        expect(wordsOf(story("attack", {region: "the world"})).caption).toMatch(/#map$/)
    })
})

describe("the call at the end of a clip", () => {
    it("shows the flag it asks to defend, which the link opens on", () => {
        const words = wordsOf(story("invasion", {country: "fr"}))

        expect(words.call).toBe("DEFEND FRANCE")
        expect(words.callFlags).toEqual(["fr"])
        expect(words.link).toBe("https://clickplanet.lol/?f=fr")
    })

    it("shows the continent's flag when it asks to defend a continent", () => {
        const words = wordsOf(story("attack", {region: "Europe"}))

        expect(words.call).toBe("DEFEND EUROPE")
        expect(words.callFlags).toEqual(["eu"])
    })

    it("shows no flag for a continent that has none", () => {
        expect(wordsOf(story("attack", {region: "Africa"})).callFlags).toEqual([])
    })

    it("shows the flag that strikes back when it asks to fight for it", () => {
        expect(wordsOf(story("comeback", {country: "ps"})).callFlags).toEqual(["ps"])
    })

    it("shows both flags when it asks to pick a side", () => {
        expect(wordsOf(story("battle", {country: "fr"}, "fr")).callFlags).toEqual(["ps", "fr"])
    })
})
