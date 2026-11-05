import {describe, expect, it} from "vitest"
import {DayMissions, Mission, MissionOrder} from "../backends/missions.ts"
import {
    MissionGround, MissionMarks, missionsOnTerritory, missionsOnTile, nearestPlayable, openTargetsOf, targetOf,
} from "./missionTargets.ts"

type Tile = {owner?: string, ground?: string, shields?: number}

function groundOf(tiles: Tile[]): MissionGround {
    return {
        ownerOf: (tile) => tiles[tile - 1]?.owner,
        shieldsOf: (tile) => tiles[tile - 1]?.shields ?? 0,
        groundOf: (tile) => tiles[tile - 1]?.ground,
    }
}

function mission(order: MissionOrder, done = false): Mission {
    return {id: order.kind, order, reward: {kind: "bomb", amount: 1}, done, paid: done}
}

describe("targetOf", () => {
    it("outlines a ground, marks a rival, and only its shielded tiles for shields to break", () => {
        expect(targetOf({kind: "takeIn", country: "kz", goal: 300, taken: 0})).toEqual({ground: "kz", shielded: false})
        expect(targetOf({kind: "bombIn", country: "kz", goal: 2, reached: 0})).toEqual({ground: "kz", shielded: false})
        expect(targetOf({kind: "takeFrom", country: "ie", goal: 80, taken: 0})).toEqual({flag: "ie", shielded: false})
        expect(targetOf({kind: "takeFromIn", country: "ie", ground: "kz", goal: 40, taken: 0}))
            .toEqual({flag: "ie", ground: "kz", shielded: false})
        expect(targetOf({kind: "breakShields", country: "ie", goal: 300, reached: 0})).toEqual({flag: "ie", shielded: true})
    })

    it("has no place for a mission played anywhere", () => {
        expect(targetOf({kind: "takeFor", country: "fr", goal: 120, taken: 0})).toBeUndefined()
        expect(targetOf({kind: "catchBoxes", goal: 3, reached: 0})).toBeUndefined()
        expect(targetOf({kind: "shieldTiles", shields: 10, goal: 2, reached: 0})).toBeUndefined()
    })
})

describe("openTargetsOf", () => {
    it("leaves out the missions done, the flag's included", () => {
        const day: DayMissions = {
            endsAt: 0,
            missions: [
                mission({kind: "takeFrom", country: "ie", goal: 80, taken: 80}, true),
                mission({kind: "takeIn", country: "kz", goal: 300, taken: 12}),
            ],
            flagMission: {id: "flag", order: {kind: "breakShields", country: "ru", goal: 30, reached: 30}, points: 5, done: true},
        }

        expect(openTargetsOf(day)).toEqual([{ground: "kz", shielded: false}])
    })
})

describe("MissionMarks", () => {
    const tiles: Tile[] = [
        {owner: "ie", ground: "kz", shields: 2},
        {owner: "ie", ground: "kz"},
        {owner: "ie", ground: "ru", shields: 1},
        {owner: "fr", ground: "kz"},
    ]

    it("marks a rival's tiles, inside its ground when it names one", () => {
        const marks = new MissionMarks(groundOf(tiles), tiles.length)

        expect(marks.follow([{flag: "ie", ground: "kz", shielded: false}]).on).toEqual([1, 2])
    })

    it("marks only the shielded ones for shields to break", () => {
        const marks = new MissionMarks(groundOf(tiles), tiles.length)

        expect(marks.follow([{flag: "ie", shielded: true}]).on).toEqual([1, 3])
    })

    it("marks no tile for a ground alone", () => {
        const marks = new MissionMarks(groundOf(tiles), tiles.length)

        expect(marks.follow([{ground: "kz", shielded: false}]).on).toEqual([])
    })

    it("unmarks a tile taken, and a shield broken", () => {
        const changing = tiles.map((tile) => ({...tile}))
        const marks = new MissionMarks(groundOf(changing), changing.length)
        marks.follow([{flag: "ie", shielded: true}])

        changing[0].owner = "fr"
        changing[2].shields = 0

        expect(marks.touch([1, 3])).toEqual({on: [], off: [1, 3]})
    })

    it("unmarks everything once no mission names a rival", () => {
        const marks = new MissionMarks(groundOf(tiles), tiles.length)
        marks.follow([{flag: "ie", shielded: false}])

        expect(marks.follow([]).off).toEqual([1, 2, 3])
    })
})

describe("nearestPlayable", () => {
    const positions = new Float32Array([1, 0, 0, 0, 1, 0, 0, 0, 1, -1, 0, 0])

    it("goes to the nearest tile of a ground the flag does not hold", () => {
        const tiles: Tile[] = [{owner: "fr", ground: "kz"}, {owner: "ie", ground: "kz"}, {ground: "kz"}, {ground: "ru"}]

        expect(nearestPlayable({ground: "kz", shielded: false}, "fr", groundOf(tiles), positions, {x: 1, y: 0.1, z: 0})).toBe(2)
    })

    it("goes to the nearest rival tile that counts", () => {
        const tiles: Tile[] = [{owner: "ie"}, {owner: "ie", shields: 1}, {owner: "fr"}, {owner: "ie", shields: 3}]

        expect(nearestPlayable({flag: "ie", shielded: true}, "fr", groundOf(tiles), positions, {x: -1, y: 0, z: 0})).toBe(4)
    })

    it("finds nothing when no tile counts", () => {
        expect(nearestPlayable({flag: "ie", shielded: false}, "fr", groundOf([{owner: "fr"}]), positions, {x: 1, y: 0, z: 0}))
            .toBeUndefined()
    })
})

describe("the missions on a tile and on a territory", () => {
    const day: DayMissions = {
        endsAt: 0,
        missions: [
            mission({kind: "takeIn", country: "kz", goal: 300, taken: 0}),
            mission({kind: "takeFromIn", country: "ie", ground: "ru", goal: 40, taken: 0}),
            mission({kind: "takeFrom", country: "pw", goal: 80, taken: 80}, true),
        ],
        flagMission: {id: "flag", order: {kind: "breakShields", country: "ie", goal: 30, reached: 0}, points: 5, done: false},
    }
    const ids = (missions: {id: string}[]) => missions.map(({id}) => id)

    it("names a mission whose ground the tile is on, and one whose ring it wears", () => {
        expect(ids(missionsOnTile(day, {owner: "fr", ground: "kz", shields: 0}))).toEqual(["takeIn"])
        expect(ids(missionsOnTile(day, {owner: "ie", ground: "ru", shields: 2}))).toEqual(["takeFromIn", "flag"])
        expect(ids(missionsOnTile(day, {owner: "ie", ground: "fr", shields: 0}))).toEqual([])
    })

    it("never names a mission done", () => {
        expect(missionsOnTile(day, {owner: "pw", ground: "pw", shields: 0})).toEqual([])
    })

    it("names a mission on a territory's ground, or on a rival holding some of it", () => {
        expect(ids(missionsOnTerritory(day, {ground: "ru", holdings: [{flag: "fr", tiles: 9}]}))).toEqual(["takeFromIn"])
        expect(ids(missionsOnTerritory(day, {ground: "gb", holdings: [{flag: "ie", tiles: 9}]}))).toEqual(["flag"])
        expect(missionsOnTerritory(day, {ground: "gb", holdings: [{flag: "fr", tiles: 9}]})).toEqual([])
    })
})
