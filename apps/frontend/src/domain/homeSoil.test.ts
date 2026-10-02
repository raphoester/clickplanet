import {describe, expect, it} from "vitest"
import {Outcome, outcomeOf, ownerAfter} from "./homeSoil.ts"

const cases: {name: string, owner?: string, ground?: string, flag: string, outcome: Outcome, after?: string}[] = [
    {name: "a native tile clicked for another flag is cleared", owner: "pl", ground: "pl", flag: "de", outcome: "cleared", after: undefined},
    {name: "a native tile clicked by its natives is unchanged", owner: "pl", ground: "pl", flag: "pl", outcome: "unchanged", after: "pl"},
    {name: "an empty tile on home ground is taken by anybody", owner: undefined, ground: "pl", flag: "de", outcome: "taken", after: "de"},
    {name: "an empty tile on home ground is taken back by its natives", owner: undefined, ground: "pl", flag: "pl", outcome: "taken", after: "pl"},
    {name: "a foreign-held tile on home ground is won back by its natives in one", owner: "de", ground: "pl", flag: "pl", outcome: "taken", after: "pl"},
    {name: "a foreign-held tile on home ground is taken by a third flag in one", owner: "de", ground: "pl", flag: "fr", outcome: "taken", after: "fr"},
    {name: "a tile already wearing the flag is unchanged", owner: "de", ground: "pl", flag: "de", outcome: "unchanged", after: "de"},
    {name: "a country's flag on another's ground is taken as always", owner: "pl", ground: "de", flag: "fr", outcome: "taken", after: "fr"},
    {name: "a tile in no country is never native", owner: "pl", ground: undefined, flag: "de", outcome: "taken", after: "de"},
    {name: "an empty tile in no country is taken", owner: undefined, ground: undefined, flag: "de", outcome: "taken", after: "de"},
]

describe("outcomeOf", () => {
    it.each(cases)("$name", ({owner, ground, flag, outcome, after}) => {
        expect(outcomeOf(owner, ground, flag)).toBe(outcome)
        expect(ownerAfter(outcome, owner, flag)).toBe(after)
    })
})
