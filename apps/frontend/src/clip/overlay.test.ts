import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {sidesOf, teamFlagOf, wordsOf} from "./overlay.ts"

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

describe("a place of two countries", () => {
    it("names both, tags both and asks to defend both", () => {
        const words = wordsOf({...story("invasion", {countries: ["eg", "tr"]}), attacker: "il"})

        expect(words.headline).toBe("ISRAEL IS INVADING EGYPT AND TURKEY")
        expect(words.callFlags).toEqual(["eg", "tr"])
        expect(words.caption).toMatch(/#map #egypt #turkey$/)
    })
})

describe("a flag thrown out of a continent", () => {
    it("is told from its side, against the continent's flag", () => {
        const words = wordsOf({...story("rout", {region: "Europe"}), attacker: "de", victims: ["ps"], team: "Europe"})

        expect(words.headline).toBe("PALESTINE GETS KICKED OUT OF EUROPE")
        expect(words.call).toBe("PICK A SIDE")
        expect(words.callFlags).toEqual(["ps", "eu"])
    })

    it("counts the continent under its flag, and the flag thrown out under its own", () => {
        const rout: Story = {...story("rout", {region: "Europe"}), attacker: "de", victims: ["ps"], team: "Europe"}

        expect(teamFlagOf(rout)).toBe("eu")
        expect(sidesOf(rout)).toEqual(["ps", "eu"])
    })
})

describe("a flag losing land nobody leads the taking of", () => {
    it("gets kicked out of a place that is not its own, and is asked to be saved", () => {
        const words = wordsOf({...story("rout", {region: "South America"}), attacker: "cy", victims: ["ps"]})

        expect(words.headline).toBe("PALESTINE GETS KICKED OUT OF SOUTH AMERICA")
        expect(words.call).toBe("FIGHT FOR PALESTINE")
        expect(words.callFlags).toEqual(["ps"])
        expect(words.caption).toMatch(/Who saves them\? 👇/)
    })

    it("is falling when the land it loses is its own country", () => {
        expect(wordsOf({...story("rout", {country: "dz"}), attacker: "bg", victims: ["dz"]}).headline).toBe("ALGERIA IS FALLING")
    })

    it("is losing a place that is its own", () => {
        expect(wordsOf({...story("rout", {region: "Africa"}), attacker: "il", victims: ["dz"]}).headline)
            .toBe("ALGERIA IS LOSING AFRICA")
    })
})

describe("names said with their article", () => {
    it("say the UK and the Netherlands, and tag them bare", () => {
        const words = wordsOf({...story("kickout", {countries: ["no", "se"]}), attacker: "gb", victims: ["il"]})

        expect(words.headline).toBe("THE UK IS KICKING ISRAEL OUT OF NORWAY AND SWEDEN")
        expect(wordsOf(story("battle", {country: "gb"}, "nl")).line).toBe("The battle for the UK")
        expect(wordsOf(story("battle", {country: "gb"}, "nl")).caption).toMatch(/#uk$/)
    })
})

describe("a continent striking back together", () => {
    it("is the continent's story, under its flag", () => {
        const words = wordsOf({...story("comeback", {region: "Europe"}), attacker: "de", team: "Europe"})

        expect(words.headline).toBe("EUROPE STRIKES BACK")
        expect(words.call).toBe("FIGHT FOR EUROPE")
        expect(words.callFlags).toEqual(["eu"])
        expect(words.caption).toMatch(/^EUROPE STRIKES BACK\. Who joins them\? 👇\n/)
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

    it("asks who joins a flag striking back, as its call asks to fight for it", () => {
        expect(wordsOf(story("comeback", {region: "Europe"})).caption).toMatch(/^PALESTINE STRIKES BACK\. Who joins them\? 👇\n/)
    })

    it("asks to pick a side in a battle", () => {
        expect(wordsOf(story("battle", {country: "fr"}, "nl")).caption).toMatch(/^PALESTINE VS THE NETHERLANDS\. Pick a side 👇\n/)
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
