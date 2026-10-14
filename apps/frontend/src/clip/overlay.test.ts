import {describe, expect, it} from "vitest"
import {sidesOf, Story, teamFlagOf, THE_WORLD} from "../domain/clip/story.ts"
import {hashtagsOf, nameOf, wordsOf} from "./overlay.ts"

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
    })

    it("is falling when the land it loses is its own country", () => {
        expect(wordsOf({...story("rout", {country: "dz"}), attacker: "bg", victims: ["dz"]}).headline).toBe("ALGERIA IS FALLING")
    })

    it("is losing a place that is its own", () => {
        expect(wordsOf({...story("rout", {region: "Africa"}), attacker: "il", victims: ["dz"]}).headline)
            .toBe("ALGERIA IS LOSING AFRICA")
    })
})

describe("the name of a country", () => {
    it("is the English name the game's was cut from", () => {
        expect(nameOf("cx")).toBe("Christmas Island")
        expect(nameOf("nz")).toBe("New Zealand")
        expect(nameOf("cz")).toBe("Czechia")
        expect(nameOf("li")).toBe("Liechtenstein")
    })

    it("is the game's when the game chose it over the English one", () => {
        expect(nameOf("tr")).toBe("Turkey")
        expect(nameOf("ci")).toBe("Ivory Coast")
        expect(nameOf("ae")).toBe("UAE")
        expect(nameOf("us")).toBe("USA")
        expect(nameOf("cd")).toBe("DR Congo")
        expect(nameOf("xb")).toBe("Brittany")
    })

    it("tells a small island taking over the world as itself", () => {
        expect(wordsOf({...story("attack", {region: "the world"}), attacker: "cx"}).headline).toBe("CHRISTMAS ISLAND IS TAKING OVER THE WORLD")
        expect(wordsOf({...story("battle", {country: "do"}, "cx")}).line).toBe("The battle for the Dominican Republic")
    })
})

describe("names said with their article", () => {
    it("say the UK and the Netherlands, and tag them bare", () => {
        const words = wordsOf({...story("kickout", {countries: ["no", "se"]}), attacker: "gb", victims: ["il"]})

        expect(words.headline).toBe("THE UK IS KICKING ISRAEL OUT OF NORWAY AND SWEDEN")
        expect(wordsOf(story("battle", {country: "gb"}, "nl")).line).toBe("The battle for the UK")
    })

    it("say the Middle East, a part of Asia, and tag it bare", () => {
        const words = wordsOf({...story("attack", {region: "Middle East"}), attacker: "dz"})

        expect(words.headline).toBe("ALGERIA IS ATTACKING THE MIDDLE EAST")
        expect(words.call).toBe("DEFEND THE MIDDLE EAST")
        expect(wordsOf({...story("comeback", {region: "Middle East"}), team: "Middle East"}).headline).toBe("THE MIDDLE EAST STRIKES BACK")
    })
})

describe("a continent striking back together", () => {
    it("is the continent's story, under its flag", () => {
        const words = wordsOf({...story("comeback", {region: "Europe"}), attacker: "de", team: "Europe"})

        expect(words.headline).toBe("EUROPE STRIKES BACK")
        expect(words.call).toBe("FIGHT FOR EUROPE")
        expect(words.callFlags).toEqual(["eu"])
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

describe("the teasers of a caption", () => {
    it("say the headline another way, as the posts written by hand did", () => {
        expect(wordsOf({...story("kickout", {country: "kz"}), attacker: "ps", victims: ["ro"]}).teasers)
            .toContain("PALESTINE TAKES KAZAKHSTAN FROM ROMANIA")
        expect(wordsOf({...story("invasion", {country: "cn"}), attacker: "hr", victims: ["fr"]}).teasers)
            .toContain("CROATIA INVADES CHINA AND KICKS OUT FRANCE")
        expect(wordsOf({...story("rout", {region: "Europe"}), attacker: "de", victims: ["cx"], team: "Europe"}).teasers)
            .toContain("GENERATIONAL DOWNFALL OF CHRISTMAS ISLAND")
        expect(wordsOf({...story("attack", {region: "Europe"}), attacker: "cx"}).teasers)
            .toContain("CHRISTMAS ISLAND TAKES OVER EUROPE")
    })

    it("tell a country throwing a flag out of its own land as taking it back", () => {
        expect(wordsOf({...story("rout", {country: "it"}), attacker: "it", victims: ["pt"]}).teasers)
            .toEqual(["ITALY RETAKES ITS LAND FROM PORTUGAL", "GENERATIONAL DOWNFALL OF PORTUGAL"])
    })

    it("name no victim an invasion did not kick out of somewhere else", () => {
        expect(wordsOf({...story("invasion", {country: "fr"}), victims: ["fr"]}).teasers)
            .toEqual(["PALESTINE INVADES FRANCE", "FRANCE IS UNDER ATTACK FROM PALESTINE"])
    })

    it("are never the headline, whatever the story", () => {
        const stories: Story[] = [
            story("invasion", {country: "tr"}), story("attack", {region: "Europe"}), story("attack", {region: "the world"}),
            story("comeback", {country: "ps"}), story("comeback", {region: "Europe"}), story("battle", {country: "fr"}, "nl"),
            {...story("kickout", {country: "au"}), attacker: "fr", victims: ["pl"]},
            {...story("rout", {region: "Europe"}), victims: ["ps"], team: "Europe"},
            {...story("rout", {country: "dz"}), victims: ["dz"]}, {...story("rout", {region: "Africa"}), victims: ["dz"]},
            {...story("comeback", {region: "Europe"}), attacker: "de", team: "Europe"},
        ]
        for (const told of stories) {
            const words = wordsOf(told)
            expect(words.teasers.length).toBeGreaterThanOrEqual(2)
            expect(words.teasers).not.toContain(words.headline)
            for (const teaser of words.teasers) expect(teaser).toBe(teaser.toUpperCase())
        }
    })

    it("keep the article of a name", () => {
        expect(wordsOf(story("battle", {country: "gb"}, "nl")).teasers).toContain("WAR FOR THE UK: PALESTINE VS THE NETHERLANDS")
    })
})

describe("the hashtags of a story", () => {
    it("name every flag of the story and the countries of its place, once each", () => {
        expect(hashtagsOf({...story("invasion", {countries: ["eg", "tr"]}), attacker: "il", victims: ["eg", "fr"]})).toEqual({
            countries: [{code: "eg", tag: "egypt"}, {code: "tr", tag: "turkey"}, {code: "il", tag: "israel"}, {code: "fr", tag: "france"}],
            continent: "africa",
        })
    })

    it("are said bare, the way people write them", () => {
        expect(hashtagsOf({...story("battle", {country: "gb"}, "nl"), victims: []}).countries.map(({tag}) => tag))
            .toEqual(["uk", "palestine", "netherlands"])
        expect(hashtagsOf({...story("attack", {region: "Middle East"}), attacker: "cx", victims: []}))
            .toEqual({countries: [{code: "cx", tag: "christmasisland"}], continent: "middleeast"})
    })

    it("leave out a continent's own flag, and the world", () => {
        expect(hashtagsOf({...story("rout", {region: THE_WORLD}), attacker: "eu", victims: ["ps"], team: "Europe"}))
            .toEqual({countries: [{code: "ps", tag: "palestine"}], continent: undefined})
    })
})

describe("a tour of the world", () => {
    it("says a flag taking land on every continent at once is taking over the world, and asks to fight back", () => {
        const words = wordsOf({...story("attack", {region: "the world"}), attacker: "pt"})

        expect(words.headline).toBe("PORTUGAL IS TAKING OVER THE WORLD")
        expect(words.call).toBe("FIGHT BACK")
        expect(words.callFlags).toEqual([])
        expect(words.link).toBe("https://clickplanet.lol")
        expect(words.teasers).toEqual(["PORTUGAL TAKES OVER THE WORLD", "NOBODY CAN STOP PORTUGAL"])
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
