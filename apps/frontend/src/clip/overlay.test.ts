import {describe, expect, it} from "vitest"
import {Day, sidesOf, Story, teamFlagOf, THE_WORLD} from "../domain/clip/story.ts"
import {hashtagsOf, nameOf, said, wordsOf, Written} from "./overlay.ts"

const day: Day = {
    season: 1, number: 5, finale: false, own: false, endedAt: 0,
    podium: [{country: "fr", rank: 1, points: 25}, {country: "pw", rank: 2, points: 18}],
    table: [{country: "fr", rank: 1, points: 86}],
}

function story(kind: Story["kind"], place: Story["place"], rival?: string): Story {
    return {kind, attacker: "ps", rival, victims: ["fr", "es"], place}
}

const written: Written = {
    headline: "CHINA GETS KICKED OUT OF FINLAND", teasers: ["China loses Finland", "CHINA GETS KICKED OUT OF FINLAND"],
    line: "Sweden's war", call: "FIGHT FOR CHINA", callFlags: ["cn"], anthem: "se",
}

describe("the words of a story the editor tells", () => {
    const rout: Story = {...story("rout", {country: "fi"}), attacker: "se", victims: ["cn"]}

    it("are the editor's alone: the headline, the line, the call and its flags", () => {
        const words = wordsOf(rout, written)

        expect(words.headline).toBe("CHINA GETS KICKED OUT OF FINLAND")
        expect(words.line).toBe("Sweden's war")
        expect(words.call).toBe("FIGHT FOR CHINA")
        expect(words.callFlags).toEqual(["cn"])
    })

    it("have teasers in capitals, named the editor's, and never the headline", () => {
        const words = wordsOf(rout, written)

        expect(words.teasers).toEqual(["CHINA LOSES FINLAND"])
        expect(words.teaserNames).toEqual(["editor"])
    })

    it("take a headline given by hand over the editor's", () => {
        expect(wordsOf(rout, written, "CHINA IS OUT").headline).toBe("CHINA IS OUT")
    })

    it("are none of the generator's own: with no words of the editor, there is no clip", () => {
        expect(() => wordsOf(rout)).toThrow("a rout is told in the editor's words, and none were given")
        expect(() => wordsOf({...story("holding", {region: "Europe"}), attacker: "hr", share: 0.8})).toThrow("none were given")
    })
})

describe("a flag thrown out of a continent", () => {
    it("counts the continent under its flag, and the flag thrown out under its own", () => {
        const rout: Story = {...story("rout", {region: "Europe"}), attacker: "de", victims: ["ps"], team: "Europe"}

        expect(teamFlagOf(rout)).toBe("eu")
        expect(sidesOf(rout)).toEqual(["ps", "eu"])
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

    it("is said with its article where English wants one", () => {
        expect(said(nameOf("gb"))).toBe("the UK")
        expect(said(nameOf("nl"))).toBe("the Netherlands")
        expect(said(nameOf("do"))).toBe("the Dominican Republic")
        expect(said("Middle East")).toBe("the Middle East")
        expect(said(nameOf("fr"))).toBe("France")
    })
})

describe("a day of the season", () => {
    it("says who won it, where that leaves it in the season, and asks who wins the next", () => {
        const words = wordsOf({...story("round", {region: "the world"}), attacker: "fr", day})

        expect(words.headline).toBe("FRANCE WINS DAY 5")
        expect(words.line).toBe("Leads the season with 86 points")
        expect(words.call).toBe("WHO WINS DAY 6?")
        expect(words.teasers).toEqual(["FRANCE TAKES DAY 5", "DAY 5 GOES TO FRANCE"])
    })

    it("places a winner behind in the season", () => {
        const behind = {...day, table: [{country: "fr", rank: 2, points: 61}, {country: "pw", rank: 1, points: 70}]}

        expect(wordsOf({...story("round", {region: "the world"}), attacker: "fr", day: behind}).line)
            .toBe("2nd in the season with 61 points")
    })

    it("crowns the winner of the final battle that leads the season", () => {
        const words = wordsOf({...story("round", {region: "the world"}), attacker: "fr", day: {...day, finale: true}})

        expect(words.headline).toBe("FRANCE WINS THE FINAL BATTLE")
        expect(words.line).toBe("Wins the season with 86 points")
        expect(words.call).toBe("PLAY FOR YOUR COUNTRY")
    })

    it("tells a battle's round of its own as the battle, never as a day", () => {
        const words = wordsOf({...story("round", {region: "the world"}), attacker: "fr", day: {...day, number: 2, own: true}})

        expect(words.headline).toBe("FRANCE WINS THE BATTLE")
        expect(words.call).toBe("PLAY FOR YOUR COUNTRY")
        expect(words.teasers).toEqual(["FRANCE TAKES THE BATTLE", "THE BATTLE GOES TO FRANCE"])
    })

    it("has teasers never its headline, each named once", () => {
        const words = wordsOf({...story("round", {region: "the world"}), attacker: "fr", day}, undefined, "FRANCE TAKES DAY 5")

        expect(words.teasers).toEqual(["DAY 5 GOES TO FRANCE"])
        expect(words.teaserNames).toEqual(["day-goes-to"])
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
