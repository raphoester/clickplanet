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
        flags: [{code: "fr", continent: "Europe", took: 5000, filled: 100, lost: 3000, tookFrom: {tr: 4800}, solidity: 0.861},
            {code: "tr", continent: "Middle East", took: 3000, filled: 0, lost: 5000, tookFrom: {fr: 3000}, solidity: 0.9}],
        bombs: 3, spreads: 0, enclosures: 1, score: 3886.52,
    }],
    tours: [{
        since: Date.UTC(2026, 9, 26, 8), until: Date.UTC(2026, 9, 26, 20), flag: "fr", took: 27000, byContinent: {Europe: 12000, "South Asia": 9000},
        tookFrom: {dz: 8000}, nextTook: 9000, solidity: 0.95, bombs: 12, flippedByContinent: {"South Asia": 6246},
    }],
    places: [{kind: "continent", place: {region: "South Asia"}, code: undefined, continent: "South Asia", tiles: 9600, holder: "fr", held: 6170, share: 0.6427, rivals: {dz: 1200}, changed: 6000}],
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
                flags: [{code: "fr", took: 5000, lost: 3000, tookFrom: {tr: 4800}, solidity: 0.86}, {code: "tr"}],
            }],
            tours: [{flag: "fr", took: 27000, byContinent: {Europe: 12000, "South Asia": 9000}, nextTook: 9000, flippedByContinent: {"South Asia": 6246}}],
            places: [{kind: "continent", continent: "South Asia", holder: "fr", share: 0.643, rivals: {dz: 1200}, changed: 6000}],
        })
    })

    it("say how every code in them is said, and name every continent with its own flag", () => {
        expect(json.names).toEqual({dz: "Algeria", fr: "France", pw: "Palau", tr: "Turkey"})
        expect(json.continents).toEqual(expect.arrayContaining([{name: "Europe", flag: "eu"}, {name: "Middle East"}, {name: "Africa"}]))
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

const WORDS = `"headline":"FRANCE TAKES INDIA","call":"FIGHT FOR ALGERIA"`

describe("the story --story names", () => {
    it("is a front with the story the editor wrote of it, in one country, two, or a continent", () => {
        const invasion: Story = {kind: "invasion", attacker: "fr", rival: undefined, victims: ["dz"], place: {country: "in"}}
        expect(toldOf(`{"front":3,"kind":"invasion","attacker":"fr","takenFrom":["dz"],"countries":["in"],${WORDS}}`).told)
            .toEqual({front: 3, story: invasion})
        expect(toldOf(`{"front":1,"kind":"battle","attacker":"fr","rival":"tr","countries":["tr","sy"],${WORDS}}`).told).toEqual({
            front: 1, story: {kind: "battle", attacker: "fr", rival: "tr", victims: [], place: {countries: ["tr", "sy"]}},
        })
        expect(toldOf(`{"front":2,"kind":"comeback","attacker":"de","takenFrom":["ps"],"continent":"Europe","team":"Europe",${WORDS}}`).told).toEqual({
            front: 2, story: {kind: "comeback", attacker: "de", rival: undefined, victims: ["ps"], place: {region: "Europe"}, team: "Europe"},
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
        })
        expect(toldOf(`{"place":4,${WORDS}}`).written).toEqual({
            headline: "FRANCE TAKES INDIA", teasers: [], line: undefined, call: "FIGHT FOR ALGERIA", callFlags: [], anthem: undefined,
        })
    })

    it("is a seasonal story by its number alone, in words of its own", () => {
        expect(toldOf(`{"seasonal":1}`)).toEqual({told: {seasonal: 1}, written: undefined})
    })

    it("is refused when it names nothing or two things, a kind no front tells, no attacker, no place, or no words", () => {
        expect(() => toldOf(`{}`)).toThrow("one front, one tour, one place or one seasonal story")
        expect(() => toldOf(`{"front":1,"tour":1}`)).toThrow("one front, one tour, one place or one seasonal story")
        expect(() => toldOf(`{"front":1,"kind":"holding","attacker":"fr","countries":["in"],${WORDS}}`)).toThrow(`not "holding"`)
        expect(() => toldOf(`{"front":1,"kind":"invasion","countries":["in"],${WORDS}}`)).toThrow("has an attacker")
        expect(() => toldOf(`{"front":1,"kind":"invasion","attacker":"fr",${WORDS}}`)).toThrow("one or two countries, or on a continent")
        expect(() => toldOf(`{"front":1,"kind":"invasion","attacker":"fr","countries":["in","pk","af"],${WORDS}}`)).toThrow("one or two countries")
        expect(() => toldOf(`{"place":4,"headline":"FRANCE HOLDS INDIA"}`)).toThrow("has its headline and its call")
        expect(() => toldOf(`{"tour":1,"call":"FIGHT BACK"}`)).toThrow("has its headline and its call")
    })
})
