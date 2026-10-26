import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {BoxPace, ClickRules, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {boxesOf, clicksOf, complete, draftOf, GROUPS, ruleSetOf, saveRuleSet, tempoOf} from "./rules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const played = () => new FakeAdmin().rules.sets[0]

const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.tempo = 3
    set.boxes = new BoxPace({minInterval: minutes(1), maxInterval: minutes(2), maxChargesPerHour: 30})
    return set
}

function leaves(json: JsonValue, path = ""): string[] {
    if (typeof json !== "object" || json === null || Array.isArray(json)) return [path]
    return Object.entries(json as JsonObject).flatMap(([key, value]) => leaves(value, path === "" ? key : `${path}.${key}`))
}

describe("the rules", () => {
    it("says the refill, the clicks and the boxes of a set", () => {
        expect(tempoOf(new RuleSet({tempo: 1}))).toBe("1×")
        expect(tempoOf(rush())).toBe("3×")
        expect(clicksOf(new ClickRules({burst: 60, perSecond: 0.2}))).toBe("60 in hand, 0.2 a second")
        expect(clicksOf(undefined)).toBe("—")
    })

    it("says how often a box comes and the hourly cap", () => {
        expect(boxesOf(rush().boxes)).toBe("Every 1 min to 2 min, 30 an hour at most")
        expect(boxesOf(new BoxPace({minInterval: minutes(4), maxInterval: minutes(4), maxChargesPerHour: 12})))
            .toBe("Every 4 min, 12 an hour at most")
        expect(boxesOf(new BoxPace({
            minInterval: new Duration({seconds: 90n}), maxInterval: minutes(2), maxChargesPerHour: 30,
        }))).toBe("Every 90 s to 2 min, 30 an hour at most")
        expect(boxesOf(undefined)).toBe("—")
    })

    it("edits every value of a set", () => {
        const edited = new Set([...GROUPS.flatMap(group => group.fields.map(field => field.path)), "name", "quiz.enabled", "tollSteps"])

        const values = leaves(played().toJson({emitDefaultValues: true}))

        expect(values.filter(path => !edited.has(path))).toEqual([])
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush())

        expect(draft.name).toBe("rush")
        expect(draft.values["tempo"]).toBe("3")
        expect(draft.values["boxes.minInterval"]).toBe("60")
        expect(draft.values["clicks.perSecond"]).toBe("0.2")
        expect(draft.values["bonuses.kinds.bomb"]).toBe("1")
        expect(draft.quiz).toBe(true)
        expect(draft.toll).toEqual([{share: "10", slowdown: "1.5"}, {share: "20", slowdown: "2.5"}, {share: "30", slowdown: "4"}])
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a toll step added, a quiz switched off and a whole count", () => {
        const draft = draftOf(rush())
        draft.toll = [...draft.toll, {share: "45", slowdown: "6"}]
        draft.quiz = false
        draft.values["bonuses.shield.perTile"] = "12.4"

        const sent = ruleSetOf(draft)

        expect(sent.tollSteps.at(-1)?.share).toBeCloseTo(0.45)
        expect(sent.tollSteps.at(-1)?.slowdown).toBe(6)
        expect(sent.quiz?.enabled).toBe(false)
        expect(sent.bonuses?.shield?.perTile).toBe(12)
    })

    it("starts a new set from the one played, with no name", () => {
        const draft = draftOf(played(), "")

        expect(complete(draft)).toBe(false)
        expect(complete({...draft, name: "rush"})).toBe(true)
        expect(ruleSetOf({...draft, name: "rush"}).clicks?.burst).toBe(60)
    })

    it("sends nothing until every value is a number and the set has a name", () => {
        const draft = draftOf(rush())

        expect(complete(draft)).toBe(true)
        expect(complete({...draft, values: {...draft.values, tempo: ""}})).toBe(false)
        expect(complete({...draft, values: {...draft.values, "quiz.leaderBias": "a lot"}})).toBe(false)
        expect(complete({...draft, toll: [{share: "10", slowdown: ""}]})).toBe(false)
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin, draftOf(rush()))

        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush())).toBe(true)
    })
})
