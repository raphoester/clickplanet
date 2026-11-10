import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {findingsOf, Found, Measured, toldOf} from "./findings.ts"

const measured: Measured = {
    world: false, since: Date.UTC(2026, 9, 24, 10), until: Date.UTC(2026, 9, 24, 12, 30), tilesChanged: 1500,
    storyTilesChanged: 1200, placeTiles: 4000, flagTiles: {start: 3000, end: 3600}, bombs: 2, spreads: 5,
    enclosures: 1, solidity: 0.9134, score: 812.345,
}

const found: Found = {
    since: Date.UTC(2026, 9, 26, 8), until: Date.UTC(2026, 9, 26, 20),
    fronts: [{
        scope: "tr", since: Date.UTC(2026, 9, 26, 15), until: Date.UTC(2026, 9, 26, 18), tilesChanged: 9000, captured: 8400,
        countries: [{code: "tr", continent: "Middle East", tiles: 2500, changed: 6000, holdersStart: {tr: 2000}, holdersEnd: {fr: 1500, tr: 900}, takers: {fr: 5000}}],
        flags: [{code: "fr", continent: "Europe", took: 5000, filled: 100, lost: 3000, tookFrom: {tr: 4800}, solidity: 0.861,
            flippedByContinent: {"Middle East": 2400},
            fighters: [{number: 1, name: "BraveFox42", took: 900}, {number: 4, name: "ŁukaszPL", took: 300}]},
        {code: "tr", continent: "Middle East", took: 3000, filled: 0, lost: 5000, tookFrom: {fr: 3000}, solidity: 0.9,
            flippedByContinent: {}, fighters: []}],
        bombs: 3, spreads: 0, enclosures: 1, score: 3886.52,
    }],
    tours: [{
        since: Date.UTC(2026, 9, 26, 8), until: Date.UTC(2026, 9, 26, 20), flag: "fr", took: 27000, byContinent: {Europe: 12000, "South Asia": 9000},
        tookFrom: {dz: 8000}, nextTook: 9000, solidity: 0.95, bombs: 12, flippedByContinent: {"South Asia": 6246},
        fighters: [{number: 1, name: "BraveFox42", took: 2100}],
    }],
    places: [{
        kind: "continent", place: {region: "South Asia"}, code: undefined, continent: "South Asia", tiles: 9600, holder: "fr", held: 6170,
        share: 0.6427, rivals: {dz: 1200}, changed: 6000, flippedByContinent: {"South Asia": 5200},
        fighters: [{number: 2, name: "StormWolf", took: 40}],
    }],
    sagas: [{
        kind: "country", place: {country: "ru"}, code: "ru", continent: "Europe", tiles: 28527, holder: "pw", share: 0.9071,
        since: Date.UTC(2026, 10, 1, 15), until: Date.UTC(2026, 10, 10, 14),
        reigns: [
            {flag: "ru", pushedAt: Date.UTC(2026, 10, 2, 22), since: Date.UTC(2026, 10, 3, 1), until: Date.UTC(2026, 10, 5, 18), peak: 0.991,
                share: 0.4, tookFrom: "ie", tookFromShare: 0.4213, lostTo: "lv"},
            {flag: "pw", pushedAt: Date.UTC(2026, 10, 7, 20), since: Date.UTC(2026, 10, 7, 21), until: undefined, peak: 0.9712,
                share: 0.9071, tookFrom: "lv", tookFromShare: 0.4404, lostTo: undefined},
        ],
        contenders: [
            {flag: "ru", since: Date.UTC(2026, 10, 1, 15), until: Date.UTC(2026, 10, 7, 21), peak: 0.991, share: 0},
            {flag: "pw", since: Date.UTC(2026, 10, 7, 20), until: undefined, peak: 0.9712, share: 0.9071},
        ],
    }],
    paths: [{flag: "pw", marks: [
        {at: Date.UTC(2026, 10, 1, 15), tilesByContinent: {}},
        {at: Date.UTC(2026, 10, 10, 14), tilesByContinent: {Europe: 25877, Oceania: 12}},
    ]}],
    continentTiles: {Europe: 40225, "Middle East": 9000},
    seasonal: [{story: {kind: "round", attacker: "pw", rival: undefined, victims: ["dz"], place: {region: "the world"},
        day: {season: 0, number: 18, finale: false, own: false, endedAt: Date.UTC(2026, 9, 25, 21), podium: [{country: "pw", rank: 1, points: 30}], table: []}}, measured}],
    anthems: ["dz", "eu", "fr"],
}

describe("the findings", () => {
    const json = findingsOf(found).toJson() as Record<string, unknown>

    it("hold every front, tour and place as measured, said in times and rounded numbers", () => {
        expect(json).toMatchObject({
            since: "2026-10-26T08:00:00Z",
            fronts: [{
                scope: "tr", since: "2026-10-26T15:00:00Z", tilesChanged: 9000, captured: 8400, score: 3886.5, bombs: 3, enclosures: 1,
                countries: [{code: "tr", continent: "Middle East", holdersEnd: {fr: 1500, tr: 900}, takers: {fr: 5000}}],
                flags: [{code: "fr", took: 5000, lost: 3000, tookFrom: {tr: 4800}, solidity: 0.86, flippedByContinent: {"Middle East": 2400}}, {code: "tr"}],
            }],
            tours: [{flag: "fr", took: 27000, byContinent: {Europe: 12000, "South Asia": 9000}, nextTook: 9000, flippedByContinent: {"South Asia": 6246}}],
            places: [{kind: "continent", continent: "South Asia", holder: "fr", share: 0.643, rivals: {dz: 1200}, changed: 6000, flippedByContinent: {"South Asia": 5200}}],
        })
    })

    it("name the top fighters of each front's flags, of each tour and of each place, by their number in the cast", () => {
        expect(json).toMatchObject({
            fronts: [{flags: [{code: "fr", fighters: [{number: 1, name: "BraveFox42", took: 900}, {number: 4, name: "ŁukaszPL", took: 300}]}, {code: "tr"}]}],
            tours: [{fighters: [{number: 1, name: "BraveFox42", took: 2100}]}],
            places: [{fighters: [{number: 2, name: "StormWolf", took: 40}]}],
        })
        expect((json.fronts as {flags: object[]}[])[0].flags[1]).not.toHaveProperty("fighters")
    })

    it("tell each place's saga: who held over half of it, from when its conquest began to when it fell", () => {
        expect(json.sagas).toEqual([{
            kind: "country", code: "ru", continent: "Europe", tiles: 28527, holder: "pw", share: 0.907,
            since: "2026-11-01T15:00:00Z", until: "2026-11-10T14:00:00Z",
            reigns: [
                {flag: "ru", pushedAt: "2026-11-02T22:00:00Z", since: "2026-11-03T01:00:00Z", until: "2026-11-05T18:00:00Z", peak: 0.991,
                    share: 0.4, tookFrom: "ie", tookFromShare: 0.421, lostTo: "lv"},
                {flag: "pw", pushedAt: "2026-11-07T20:00:00Z", since: "2026-11-07T21:00:00Z", peak: 0.971, share: 0.907, tookFrom: "lv",
                    tookFromShare: 0.44},
            ],
            contenders: [
                {flag: "ru", since: "2026-11-01T15:00:00Z", until: "2026-11-07T21:00:00Z", peak: 0.991},
                {flag: "pw", since: "2026-11-07T20:00:00Z", peak: 0.971, share: 0.907},
            ],
        }])
    })

    it("tell each reigning flag's land on each continent over the season, and each continent's size", () => {
        expect(json.paths).toEqual([{flag: "pw", marks: [
            {at: "2026-11-01T15:00:00Z"},
            {at: "2026-11-10T14:00:00Z", tilesByContinent: {Europe: 25877, Oceania: 12}},
        ]}])
        expect(json.continents).toEqual(expect.arrayContaining([{name: "Europe", flag: "eu", tiles: 40225}, {name: "Middle East", tiles: 9000}]))
    })

    it("say how every code in them is said, and name every continent with its own flag", () => {
        expect(json.names).toEqual({dz: "Algeria", fr: "France", ie: "Ireland", lv: "Latvia", pw: "Palau", ru: "Russia", tr: "Turkey"})
        expect(json.continents).toEqual(expect.arrayContaining([{name: "Europe", flag: "eu", tiles: 40225}, {name: "Middle East", tiles: 9000}, {name: "Africa"}]))
    })

    it("list the anthems a clip can play", () => {
        expect(json.anthems).toEqual(["dz", "eu", "fr"])
    })

    it("tell a seasonal story whole: its headline, its key, its countries, its teasers and its facts", () => {
        expect(json).toMatchObject({
            seasonal: [{
                title: "PALAU WINS DAY 18", key: {flag: "pw", continent: "the world", angle: "round", round: "0.18"}, countries: ["pw", "dz"],
                facts: {kind: "round", place: "the world", camera: "front", attacker: "pw", takenFrom: ["dz"], spanMinutes: 150, flagTilesStart: 3000, solidity: 0.91},
            }],
        })
        expect((json.seasonal as {teasers: string[]}[])[0].teasers).not.toContain("PALAU WINS DAY 18")
    })
})

const WORDS = `"headline":"FRANCE TAKES INDIA","call":"FIGHT FOR ALGERIA","look":"dive"`

describe("the story --story names", () => {
    it("is a front with the story the editor wrote of it, in one country, several, or a continent", () => {
        const invasion: Story = {kind: "invasion", attacker: "fr", rival: undefined, victims: ["dz"], place: {country: "in"}}
        expect(toldOf(`{"front":3,"kind":"invasion","attacker":"fr","takenFrom":["dz"],"countries":["in"],${WORDS}}`).told)
            .toEqual({front: 3, story: invasion, look: "dive"})
        expect(toldOf(`{"front":1,"kind":"battle","attacker":"fr","rival":"tr","countries":["tr","sy","iq"],${WORDS}}`).told).toEqual({
            front: 1, story: {kind: "battle", attacker: "fr", rival: "tr", victims: [], place: {countries: ["tr", "sy", "iq"]}}, look: "dive",
        })
        expect(toldOf(`{"front":2,"kind":"comeback","attacker":"de","takenFrom":["ps"],"continent":"Europe","team":"Europe",${WORDS}}`).told).toEqual({
            front: 2, story: {kind: "comeback", attacker: "de", rival: undefined, victims: ["ps"], place: {region: "Europe"}, team: "Europe"},
            look: "dive",
        })
    })

    it("carries every word the editor wrote, and its anthem", () => {
        expect(toldOf(`{"tour":1,"headline":"FRANCE IS EVERYWHERE","teasers":["NOBODY CAN STOP FRANCE"],"line":"Europe, then India",`
            + `"call":"FIGHT BACK","callFlags":["eu"],"anthem":"fr"}`)).toEqual({
            told: {tour: 1},
            written: {
                headline: "FRANCE IS EVERYWHERE", teasers: ["NOBODY CAN STOP FRANCE"], line: "Europe, then India", call: "FIGHT BACK",
                callFlags: ["eu"], anthem: "fr",
            },
            fighters: [],
        })
        expect(toldOf(`{"place":4,${WORDS}}`).written).toEqual({
            headline: "FRANCE TAKES INDIA", teasers: [], line: undefined, call: "FIGHT FOR ALGERIA", callFlags: [], anthem: undefined,
        })
    })

    it("carries the look the editor chose for a front or a place; a tour has none", () => {
        expect(toldOf(`{"place":4,"headline":"FRANCE HOLDS INDIA","call":"FIGHT BACK","look":"flags"}`).told).toEqual({place: 4, look: "flags"})
        expect(toldOf(`{"tour":1,"headline":"FRANCE IS EVERYWHERE","call":"FIGHT BACK"}`).told).toEqual({tour: 1})
    })

    it("names the players of the cast the editor chose, by number, but for a seasonal story, which names nobody", () => {
        expect(toldOf(`{"front":2,"kind":"battle","attacker":"fr","rival":"tr","countries":["tr"],${WORDS},"fighters":[3,1,7]}`).fighters).toEqual([3, 1, 7])
        expect(toldOf(`{"tour":1,${WORDS},"fighters":[2]}`).fighters).toEqual([2])
        expect(toldOf(`{"place":1,${WORDS},"fighters":[5]}`).fighters).toEqual([5])
        expect(toldOf(`{"seasonal":1,"fighters":[1]}`).fighters).toEqual([])
    })

    it("is a saga told from a time, about one of its flags, against another or none, with no look", () => {
        const since = Date.UTC(2026, 10, 7, 19)
        expect(toldOf(`{"saga":2,"since":"2026-11-07T19:00:00Z","attacker":"de","rival":"cz","headline":"GERMANY TAKES CHINA","call":"FIGHT BACK"}`).told)
            .toEqual({saga: 2, since, attacker: "de", rival: "cz"})
        expect(toldOf(`{"saga":1,"since":"2026-11-07T19:00:00Z","attacker":"pw","headline":"PALAU CONQUERS RUSSIA","call":"FIGHT BACK","fighters":[3]}`))
            .toMatchObject({told: {saga: 1, since, attacker: "pw", rival: undefined}, fighters: []})
        expect(() => toldOf(`{"saga":1,"attacker":"pw",${WORDS}}`)).toThrow("from a time, about one of its flags")
        expect(() => toldOf(`{"saga":1,"since":"2026-11-07T19:00:00Z",${WORDS}}`)).toThrow("from a time, about one of its flags")
    })

    it("is a seasonal story by its number alone, in words of its own", () => {
        expect(toldOf(`{"seasonal":1}`)).toEqual({told: {seasonal: 1}, written: undefined, fighters: []})
    })

    it("is refused when it names nothing or two things, a kind no front tells, no attacker, no place, no words or no look", () => {
        expect(() => toldOf(`{}`)).toThrow("one front, one tour, one place, one saga or one seasonal story")
        expect(() => toldOf(`{"front":1,"tour":1}`)).toThrow("one front, one tour, one place, one saga or one seasonal story")
        expect(() => toldOf(`{"front":1,"kind":"holding","attacker":"fr","countries":["in"],${WORDS}}`)).toThrow(`not "holding"`)
        expect(() => toldOf(`{"front":1,"kind":"saga","attacker":"fr","countries":["in"],${WORDS}}`)).toThrow(`not "saga"`)
        expect(() => toldOf(`{"front":1,"kind":"invasion","countries":["in"],${WORDS}}`)).toThrow("has an attacker")
        expect(() => toldOf(`{"front":1,"kind":"invasion","attacker":"fr",${WORDS}}`)).toThrow("in countries, or on a continent")
        expect(() => toldOf(`{"place":4,"headline":"FRANCE HOLDS INDIA"}`)).toThrow("has its headline and its call")
        expect(() => toldOf(`{"tour":1,"call":"FIGHT BACK"}`)).toThrow("has its headline and its call")
        expect(() => toldOf(`{"place":4,"headline":"FRANCE HOLDS INDIA","call":"FIGHT BACK"}`)).toThrow(`not ""`)
        expect(() => toldOf(`{"front":1,"kind":"invasion","attacker":"fr","countries":["in"],"headline":"FRANCE TAKES INDIA",`
            + `"call":"FIGHT BACK","look":"tour"}`)).toThrow(`not "tour"`)
    })
})
