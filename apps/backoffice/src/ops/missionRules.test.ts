import {describe, expect, it} from "vitest"
import {FlagMissionRules, Rules} from "../gen/grpc/missions/v1/rules_pb.ts"
import {changesOf, draftOf, FIELDS, flawOf, rulesOf} from "./missionRules.ts"

const kept = Rules.fromJson({flagMissions: {goalPercent: 30, leastGoal: 250, fewestPoints: 3, mostPoints: 10}})

describe("the mission rules form", () => {
    it("edits every number of a flag mission", () => {
        expect(FIELDS.map(field => field.key).sort()).toEqual(FlagMissionRules.fields.list().map(field => field.localName).sort())
    })

    it("sends back what it read", () => {
        expect(rulesOf(draftOf(kept)).equals(kept)).toBe(true)
    })

    it("says what a save changes", () => {
        const draft = {...draftOf(kept), goalPercent: " 40 "}

        expect(changesOf(draftOf(kept), draft)).toEqual([
            {label: "Goal, % of all the flag did the day before", before: "30", after: "40"},
        ])
        expect(rulesOf(draft).flagMissions?.goalPercent).toBe(40)
    })

    it("names a number left empty or not whole", () => {
        expect(flawOf(draftOf(kept))).toBeUndefined()
        expect(flawOf({...draftOf(kept), leastGoal: ""})).toBe("Least goal, tiles or shields: a whole number.")
        expect(flawOf({...draftOf(kept), mostPoints: "2.5"})).toBe("Points for a flag holding no tile: a whole number.")
    })
})
