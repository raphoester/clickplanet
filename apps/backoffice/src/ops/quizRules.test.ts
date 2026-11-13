import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/quizzes/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/quizzes/v1/rules_pb.ts"
import {
    changesOf, complete, copyOf, draftOf, FROZEN, GROUPS, offersOf, playedOf, quizzesOf, ruleSetOf, saveRuleSet,
} from "./quizRules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const played = () => new FakeAdmin().quizRules.sets[0]

// A quiz every two or three minutes, twenty right an hour.
const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.rules!.minInterval = minutes(2)
    set.rules!.maxInterval = minutes(3)
    set.rules!.maxRightAnswersPerHour = 20
    return set
}

function leaves(json: JsonValue, path = ""): string[] {
    if (typeof json !== "object" || json === null || Array.isArray(json)) return [path]
    return Object.entries(json as JsonObject).flatMap(([key, value]) => leaves(value, path === "" ? key : `${path}.${key}`))
}

describe("the quiz rules", () => {
    it("says whether a set asks, and how often a quiz comes", () => {
        expect(offersOf(played())).toBe("Open")
        expect(offersOf(new RuleSet({rules: Rules.fromJson({frozen: true})}))).toBe("Frozen")
        expect(quizzesOf(played().rules)).toBe("Every 4 min to 8 min, 12 right an hour at most")
        expect(quizzesOf(rush().rules)).toBe("Every 2 min to 3 min, 20 right an hour at most")
        expect(quizzesOf(Rules.fromJson({enabled: true, minInterval: "90s", maxInterval: "90s", maxRightAnswersPerHour: 3})))
            .toBe("Every 90 s, 3 right an hour at most")
        expect(quizzesOf(Rules.fromJson({enabled: false, minInterval: "60s"}))).toBe("Off")
        expect(quizzesOf(new Rules())).toBe("Off")
        expect(quizzesOf(undefined)).toBe("—")
    })

    it("edits every value of a set", () => {
        const edited = new Set([
            FROZEN.path, ...GROUPS.flatMap(group => [...group.switches, ...group.fields].map(each => each.path)),
        ])

        const values = leaves(played().rules!.toJson({emitDefaultValues: true}))

        expect(values.filter(path => !edited.has(path))).toEqual([])
        expect([...edited].filter(path => !values.includes(path))).toEqual([])
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush())

        expect(draft.name).toBe("rush")
        expect(draft.values["minInterval"]).toBe("120")
        expect(draft.values["maxRightAnswersPerHour"]).toBe("20")
        expect(draft.values["answerWindow"]).toBe("8")
        expect(draft.values["width"]).toBe("1")
        expect(draft.values["welcome.within"]).toBe("600")
        expect(draft.switches).toEqual({"frozen": false, "enabled": true})
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a whole count, a duration in seconds and a set frozen", () => {
        const draft = draftOf(rush())
        draft.values["floorAfter"] = "6.4"
        draft.values["answerWindow"] = "5"
        draft.switches.frozen = true

        const sent = ruleSetOf(draft).rules

        expect(sent?.floorAfter).toBe(6)
        expect(Number(sent?.answerWindow?.seconds)).toBe(5)
        expect(sent?.frozen).toBe(true)
        expect(offersOf(ruleSetOf(draft))).toBe("Frozen")
    })

    it("copies default as a set with no name that holds nothing, so it plays as default", () => {
        const draft = copyOf(played())

        expect(complete(draft)).toBe(false)
        expect(complete({...draft, name: "rush"})).toBe(true)
        const sent = ruleSetOf({...draft, name: "rush"})
        expect(sent.rules?.equals(new Rules())).toBe(true)
        expect(playedOf(sent, played()).rules?.maxRightAnswersPerHour).toBe(12)
    })

    it("copies another set with what it holds", () => {
        const quick = new RuleSet({name: "quick", rules: Rules.fromJson({minInterval: "60s", enabled: false})})

        const draft = copyOf(quick)

        expect(draft.name).toBe("")
        expect(draft.values["minInterval"]).toBe("60")
        expect(draft.values["maxInterval"]).toBe("")
        expect(draft.switches.enabled).toBe(false)
        expect(draft.switches.frozen).toBeUndefined()
    })

    it("sends only the values a set holds, zeros and noes included", () => {
        const draft = draftOf(new RuleSet({name: "quiet"}))
        draft.values["width"] = "0"
        draft.values["welcome.within"] = "0"
        draft.switches.enabled = false

        expect(ruleSetOf(draft).rules?.toJson()).toEqual({enabled: false, width: 0, welcome: {within: "0s"}})
    })

    it("plays default's values where a set holds none", () => {
        const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({maxRightAnswersPerHour: 6, enabled: false})})

        const rules = playedOf(quiet, played()).rules

        expect(rules?.maxRightAnswersPerHour).toBe(6)
        expect(rules?.enabled).toBe(false)
        expect(Number(rules?.minInterval?.seconds)).toBe(240)
        expect(rules?.target).toBe(0.7)
        expect(Number(rules?.welcome?.within?.seconds)).toBe(600)
        expect(playedOf(played(), played()).equals(played())).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name; default needs every value", () => {
        const draft = draftOf(rush())

        expect(complete(draft)).toBe(true)
        expect(complete({...draft, values: {...draft.values, "offerTtl": ""}})).toBe(true)
        expect(complete({...draftOf(played()), values: {...draftOf(played()).values, "offerTtl": ""}})).toBe(false)
        expect(complete({...draft, values: {...draft.values, "leaderBias": "a lot"}})).toBe(false)
        expect(complete({...draft, name: " "})).toBe(false)
    })

    it("says what changes, in the form's order, durations in the unit they read best in", () => {
        const before = draftOf(rush())
        const after = {
            ...before,
            values: {...before.values, "minInterval": "60", "answerWindow": "", "target": "0.70", "explore": "0.2"},
            switches: {...before.switches, "frozen": true, "enabled": undefined},
        }

        expect(changesOf(before, after)).toEqual([
            {label: "Frozen", before: "no", after: "yes"},
            {label: "Quizzes on", before: "yes", after: "default"},
            {label: "Quizzes · Shortest quiz wait (s)", before: "2 min", after: "1 min"},
            {label: "Quizzes · Time to answer (s)", before: "8 s", after: "default"},
            {label: "Quiz difficulty · Draws that ignore it, share", before: "0.1", after: "0.2"},
        ])
    })

    it("saves a set as the form holds it, to the quizzes", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin.quizzes, {...draftOf(rush()), name: " rush "})

        const [asked] = admin.askedFor("saveQuizRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush())).toBe(true)
        expect(admin.askedFor("saveBonusRuleSet")).toEqual([])
    })
})
