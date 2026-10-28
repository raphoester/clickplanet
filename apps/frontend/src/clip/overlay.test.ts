// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import {NamedSide} from "../domain/clip/cast.ts"
import {Day, sidesOf, Story, teamFlagOf, THE_WORLD} from "../domain/clip/story.ts"
import {createOverlay, hashtagsOf, nameOf, wordsOf} from "./overlay.ts"

const day: Day = {
    season: 1, number: 5, finale: false, own: false, endedAt: 0,
    podium: [{country: "fr", rank: 1, points: 25}, {country: "pw", rank: 2, points: 18}],
    table: [{country: "fr", rank: 1, points: 86}],
}

function story(kind: Story["kind"], place: Story["place"], rival?: string): Story {
    return {kind, attacker: "ps", rival, victims: ["fr", "es"], place}
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
            {...story("holding", {region: "Europe"}), attacker: "hr", share: 0.8},
            {...story("holding", {region: "the world"}), share: 0.12}, {...story("holding", {country: "br"}), share: 0.92},
            {...story("round", {region: "the world"}), attacker: "fr", day},
            {...story("round", {region: "the world"}), attacker: "fr", day: {...day, finale: true}},
        ]
        for (const told of stories) {
            const words = wordsOf(told)
            expect(words.teasers.length).toBeGreaterThanOrEqual(2)
            expect(words.teasers).not.toContain(words.headline)
            for (const teaser of words.teasers) expect(teaser).toBe(teaser.toUpperCase())
            expect(words.teaserNames).toHaveLength(words.teasers.length)
            expect(new Set(words.teaserNames).size).toBe(words.teaserNames.length)
            for (const name of words.teaserNames) expect(name).toMatch(/^[a-z]+(-[a-z]+)*$/)
        }
    })

    it("name each wording the same whatever the flags, so the posts can be compared by it", () => {
        const portugal = wordsOf({...story("invasion", {country: "fr"}), attacker: "pt", victims: ["fr"]})
        const palestine = wordsOf({...story("invasion", {country: "tr"}), victims: ["tr"]})

        expect(portugal.teaserNames).toEqual(["invades", "under-attack-from"])
        expect(palestine.teaserNames).toEqual(portugal.teaserNames)
    })

    it("keep the article of a name", () => {
        expect(wordsOf(story("battle", {country: "gb"}, "nl")).teasers).toContain("WAR FOR THE UK: PALESTINE VS THE NETHERLANDS")
    })
})

describe("what a flag holds", () => {
    it("says the share of the place it holds, never more than it holds, and asks to take it back", () => {
        const words = wordsOf({...story("holding", {region: "Europe"}), attacker: "hr", victims: ["se"], share: 0.7996})

        expect(words.headline).toBe("CROATIA HOLDS 79% OF EUROPE")
        expect(words.line).toBeUndefined()
        expect(words.call).toBe("TAKE EUROPE BACK")
        expect(words.callFlags).toEqual(["eu"])
        expect(words.teasers).toEqual(["79% OF EUROPE FLIES THE FLAG OF CROATIA", "CROATIA RULES EUROPE"])
    })

    it("calls a flag holding less than half of a place its biggest power, which it is", () => {
        expect(wordsOf({...story("holding", {region: "Africa"}), attacker: "fr", share: 0.42}).teasers)
            .toContain("FRANCE IS THE BIGGEST POWER IN AFRICA")
    })

    it("asks a country's players to take it back from a foreign flag", () => {
        const words = wordsOf({...story("holding", {country: "br"}), attacker: "pt", share: 0.92})

        expect(words.headline).toBe("PORTUGAL HOLDS 92% OF BRAZIL")
        expect(words.callFlags).toEqual(["br"])
        expect(words.link).toBe("https://clickplanet.lol/?f=br")
    })

    it("calls the world's biggest holder the biggest empire on earth, and asks everyone to fight back", () => {
        const words = wordsOf({...story("holding", {region: "the world"}), attacker: "pw", share: 0.124})

        expect(words.headline).toBe("PALAU HOLDS 12% OF THE WORLD")
        expect(words.call).toBe("FIGHT BACK")
        expect(words.callFlags).toEqual([])
        expect(words.teasers).toContain("THE BIGGEST EMPIRE ON EARTH: PALAU")
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

describe("the end card of the players a clip names", () => {
    const battle = story("battle", {country: "fr"}, "nl")
    const named: NamedSide[] = [
        {flag: "ps", fighters: [{number: 1, name: "BraveFox42", took: 1234}, {number: 3, name: "Toto", took: 1}]},
        {flag: "nl", fighters: [{number: 2, name: "StormWolf", took: 40}]},
    ]
    const overlaid = (sides: NamedSide[]) => {
        const root = document.createElement("div")
        return {root, overlay: createOverlay(root, battle, wordsOf(battle), new Map(), sides)}
    }

    it("lists each side's players under its flag with the tiles each took, the sides apart", () => {
        const {root} = overlaid(named)

        expect(root.querySelector(".clip-ending .clip-fighters-title")?.textContent).toBe("TOP FIGHTERS")
        expect([...root.querySelectorAll(".clip-fighters-side")].map((side) => [...side.querySelectorAll(".clip-fighter")].map((row) => [
            row.querySelector(".clip-fighter-flag")?.getAttribute("src"),
            row.querySelector(".clip-fighter-name")?.textContent,
            row.querySelector(".clip-fighter-took")?.textContent,
        ]))).toEqual([
            [["/static/countries/svg/ps.svg", "BraveFox42", "1,234 tiles"], ["/static/countries/svg/ps.svg", "Toto", "1 tile"]],
            [["/static/countries/svg/nl.svg", "StormWolf", "40 tiles"]],
        ])
    })

    it("shows in the call to act once the counter is gone, never over it", () => {
        const {root, overlay} = overlaid(named)
        const opacities = (ending: number) => {
            overlay.show({replayAt: 0, played: 1, held: new Map(), ending})
            return [".clip-bottom", ".clip-ending", ".clip-fighters"].map((part) => root.querySelector<HTMLElement>(part)?.style.opacity)
        }

        expect(opacities(-1)).toEqual(["1", "0", "0"])
        expect(opacities(0.25)).toEqual(["0.5", "0.5", "0"])
        expect(opacities(0.5)).toEqual(["0", "1", "0"])
        expect(opacities(0.75)).toEqual(["0", "1", "0.5"])
        expect(opacities(2)).toEqual(["0", "1", "1"])
    })

    it("sets a name too long for its row smaller to fit it, never cut nor wrapped", () => {
        const {root, overlay} = overlaid(named)
        const [long, short] = root.querySelectorAll<HTMLElement>(".clip-fighter-name")
        const measured = (name: HTMLElement, clientWidth: number, scrollWidth: number) => {
            Object.defineProperty(name, "clientWidth", {value: clientWidth})
            Object.defineProperty(name, "scrollWidth", {value: scrollWidth})
        }
        measured(long, 200, 300)
        measured(short, 200, 120)

        overlay.fit()

        expect([long.style.fontSize, short.style.fontSize]).toEqual(["0.66em", ""])
    })

    it("is left out when the clip names nobody", () => {
        expect(overlaid([]).root.querySelector(".clip-fighters")).toBeNull()
    })

    it("sets the whole clip in Rubik for a name Luckiest Guy cannot draw, and only then", () => {
        const named = (name: string) => [{flag: "ps", fighters: [{number: 1, name, took: 12}]}]

        expect(overlaid(named("مقاتل_7")).root.classList.contains("clip--fallback-font")).toBe(true)
        expect(overlaid(named("ŁukaszPL")).root.classList.contains("clip--fallback-font")).toBe(true)
        expect(overlaid(named("Jérôme_99")).root.classList.contains("clip--fallback-font")).toBe(false)
    })
})
