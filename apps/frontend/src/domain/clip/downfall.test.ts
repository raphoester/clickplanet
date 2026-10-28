import {describe, expect, it} from "vitest"
import {downfallOf} from "./downfall.ts"
import {Story} from "./story.ts"

function world(...holdings: [string, number][]): Map<number, string> {
    const owners = new Map<number, string>()
    for (const [flag, count] of holdings) {
        for (let i = 0; i < count; i++) owners.set(owners.size + 1, flag)
    }
    return owners
}

const OTHERS: [string, number][] = ["de", "fr", "it", "es", "pt", "pl", "ro", "nl", "be", "at", "ch"]
    .map((flag, i) => [flag, 1000 - i * 50])

function rout(loser: string): Story {
    return {kind: "rout", attacker: "se", rival: undefined, victims: [loser], place: {country: "fi"}}
}

describe("a generational downfall", () => {
    it("is a flag at the top of the world falling far down it", () => {
        const opening = world(["cn", 5000], ...OTHERS)
        const end = world(["cn", 100], ...OTHERS)

        expect(downfallOf(rout("cn"), opening, end)).toBe(true)
    })

    it("is never one place lost by a flag that held little", () => {
        const opening = world(...OTHERS, ["cn", 300])
        const end = world(...OTHERS, ["cn", 20])

        expect(downfallOf(rout("cn"), opening, end)).toBe(false)
    })

    it("is never a leader losing much and staying near the top", () => {
        const opening = world(["cn", 5000], ...OTHERS)
        const end = world(["cn", 900], ...OTHERS)

        expect(downfallOf(rout("cn"), opening, end)).toBe(false)
    })

    it("is only ever a rout", () => {
        const opening = world(["cn", 5000], ...OTHERS)
        const end = world(["cn", 100], ...OTHERS)

        expect(downfallOf({...rout("cn"), kind: "battle", attacker: "cn", rival: "se"}, opening, end)).toBe(false)
    })
})
