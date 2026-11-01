import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {BoxPace, ClickRules, Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {
    boxesOf, changesOf, clicksOf, complete, copyOf, draftOf, GROUPS, playedOf, playOf, ruleSetOf, saveRuleSet, tempoOf,
} from "./rules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const played = () => new FakeAdmin().rules.sets[0]

const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.rules!.clicks!.tempo = 3
    set.rules!.bonuses!.boxes = new BoxPace({minInterval: minutes(1), maxInterval: minutes(2), maxChargesPerHour: 30})
    return set
}

function leaves(json: JsonValue, path = ""): string[] {
    if (typeof json !== "object" || json === null || Array.isArray(json)) return [path]
    return Object.entries(json as JsonObject).flatMap(([key, value]) => leaves(value, path === "" ? key : `${path}.${key}`))
}

describe("the rules", () => {
    it("says the refill, the clicks and the boxes of a set", () => {
        expect(tempoOf(played())).toBe("1×")
        expect(tempoOf(rush())).toBe("3×")
        expect(tempoOf(new RuleSet())).toBe("0×")
        expect(clicksOf(new ClickRules({burst: 60, perSecond: 0.2}))).toBe("60 in hand, 0.2 a second")
        expect(clicksOf(undefined)).toBe("—")
    })

    it("says how often a box comes and the hourly cap", () => {
        expect(boxesOf(rush().rules?.bonuses?.boxes)).toBe("Every 1 min to 2 min, 30 an hour at most")
        expect(boxesOf(new BoxPace({minInterval: minutes(4), maxInterval: minutes(4), maxChargesPerHour: 12})))
            .toBe("Every 4 min, 12 an hour at most")
        expect(boxesOf(new BoxPace({
            minInterval: new Duration({seconds: 90n}), maxInterval: minutes(2), maxChargesPerHour: 30,
        }))).toBe("Every 90 s to 2 min, 30 an hour at most")
        expect(boxesOf(undefined)).toBe("—")
    })

    it("edits every value of a set", () => {
        const edited = new Set([...GROUPS.flatMap(group => group.fields.map(field => field.path)), "bonuses.quiz.enabled", "bonuses.signInRefill", "toll.steps", "frozen", "allegianceLocked"])

        const values = leaves(played().rules!.toJson({emitDefaultValues: true}))

        expect(values.filter(path => !edited.has(path))).toEqual([])
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush())

        expect(draft.name).toBe("rush")
        expect(draft.values["clicks.tempo"]).toBe("3")
        expect(draft.values["bonuses.boxes.minInterval"]).toBe("60")
        expect(draft.values["clicks.perSecond"]).toBe("0.2")
        expect(draft.values["bonuses.kinds.bomb"]).toBe("1")
        expect(draft.quiz).toBe(true)
        expect(draft.toll).toEqual([{share: "10", slowdown: "1.5"}, {share: "20", slowdown: "2.5"}, {share: "30", slowdown: "4"}])
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a toll step added, a quiz switched off and a whole count", () => {
        const draft = draftOf(rush())
        draft.toll = [...draft.toll ?? [], {share: "45", slowdown: "6"}]
        draft.quiz = false
        draft.values["bonuses.shield.perTile"] = "12.4"

        const sent = ruleSetOf(draft).rules

        expect(sent?.toll?.steps.at(-1)?.share).toBeCloseTo(0.45)
        expect(sent?.toll?.steps.at(-1)?.slowdown).toBe(6)
        expect(sent?.bonuses?.quiz?.enabled).toBe(false)
        expect(sent?.bonuses?.shield?.perTile).toBe(12)
    })

    it("sends the refill at first sign-in switched off, and says it changed", () => {
        const before = draftOf(played())
        const after = {...before, signInRefill: false}

        expect(before.signInRefill).toBe(true)
        expect(ruleSetOf(after).rules?.bonuses?.signInRefill).toBe(false)
        expect(changesOf(before, after)).toEqual([{label: "Refill at first sign-in", before: "yes", after: "no"}])
    })

    it("freezes a set, and says so in the list", () => {
        const draft = copyOf(rush())
        draft.name = "frozen"
        draft.frozen = true

        const sent = ruleSetOf(draft)

        expect(sent.rules?.frozen).toBe(true)
        expect(playOf(sent)).toBe("Frozen")
        expect(playOf(rush())).toBe("Open")
        expect(draftOf(sent).frozen).toBe(true)
    })

    it("copies default as a set with no name that holds nothing, so it plays as default", () => {
        const draft = copyOf(played())

        expect(complete(draft)).toBe(false)
        expect(complete({...draft, name: "rush"})).toBe(true)
        const sent = ruleSetOf({...draft, name: "rush"})
        expect(sent.rules?.equals(new Rules())).toBe(true)
        expect(playedOf(sent, played()).rules?.clicks?.burst).toBe(60)
    })

    it("copies another set with what it holds", () => {
        const quick = new RuleSet({name: "quick", rules: Rules.fromJson({clicks: {tempo: 3}, frozen: true})})

        const draft = copyOf(quick)

        expect(draft.name).toBe("")
        expect(draft.values["clicks.tempo"]).toBe("3")
        expect(draft.frozen).toBe(true)
        expect(draft.quiz).toBeUndefined()
    })

    it("sends only the values a set holds, zeros and an empty toll included", () => {
        const draft = draftOf(new RuleSet({name: "quiet"}))
        draft.values["clicks.newAccountClicks"] = "0"
        draft.quiz = false
        draft.toll = []

        expect(ruleSetOf(draft).rules?.toJson()).toEqual({clicks: {newAccountClicks: 0}, toll: {}, bonuses: {quiz: {enabled: false}}})
    })

    it("plays default's values where a set holds none, and a toll it holds whole", () => {
        const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({clicks: {tempo: 3}, bonuses: {quiz: {enabled: false}}, toll: {}})})

        const rules = playedOf(quiet, played()).rules

        expect(rules?.clicks?.tempo).toBe(3)
        expect(rules?.clicks?.burst).toBe(60)
        expect(rules?.bonuses?.quiz?.enabled).toBe(false)
        expect(rules?.bonuses?.quiz?.maxChargesPerHour).toBe(12)
        expect(rules?.toll?.steps).toEqual([])
        expect(playedOf(played(), played()).equals(played())).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name; default needs every value", () => {
        const draft = draftOf(rush())

        expect(complete(draft)).toBe(true)
        expect(complete({...draft, values: {...draft.values, "clicks.tempo": ""}})).toBe(true)
        expect(complete({...draftOf(played()), values: {...draftOf(played()).values, "clicks.tempo": ""}})).toBe(false)
        expect(complete({...draft, values: {...draft.values, "bonuses.quiz.leaderBias": "a lot"}})).toBe(false)
        expect(complete({...draft, toll: [{share: "10", slowdown: ""}]})).toBe(false)
    })

    it("says a value that follows default", () => {
        const before = draftOf(rush())
        const after = {...before, values: {...before.values, "clicks.tempo": ""}, quiz: undefined, toll: undefined}

        expect(changesOf(before, after)).toEqual([
            {label: "Quizzes on", before: "yes", after: "default"},
            {label: "Clicks · Refill, times as fast", before: "3", after: "default"},
            {label: "Toll", before: "10% ×1.5, 20% ×2.5, 30% ×4", after: "default"},
        ])
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin, draftOf(rush()))

        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush())).toBe(true)
    })
})
