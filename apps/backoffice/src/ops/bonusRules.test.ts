import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {BoxPace, Rules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import {
    boxesOf, changesOf, complete, copyOf, draftOf, FROZEN, GROUPS, offersOf, playedOf, ruleSetOf, saveRuleSet,
} from "./bonusRules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const played = () => new FakeAdmin().bonusRules.sets[0]

// Boxes every minute or two, thirty an hour.
const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.rules!.boxes = new BoxPace({minInterval: minutes(1), maxInterval: minutes(2), maxChargesPerHour: 30})
    return set
}

function leaves(json: JsonValue, path = ""): string[] {
    if (typeof json !== "object" || json === null || Array.isArray(json)) return [path]
    return Object.entries(json as JsonObject).flatMap(([key, value]) => leaves(value, path === "" ? key : `${path}.${key}`))
}

describe("the box rules", () => {
    it("says whether a set offers, and how often a box comes", () => {
        expect(offersOf(played())).toBe("Open")
        expect(offersOf(new RuleSet({rules: Rules.fromJson({frozen: true})}))).toBe("Frozen")
        expect(boxesOf(played().rules?.boxes)).toBe("Every 4 min to 8 min, 12 an hour at most")
        expect(boxesOf(rush().rules?.boxes)).toBe("Every 1 min to 2 min, 30 an hour at most")
        expect(boxesOf(new BoxPace({
            minInterval: new Duration({seconds: 90n}), maxInterval: new Duration({seconds: 90n}), maxChargesPerHour: 30,
        }))).toBe("Every 90 s, 30 an hour at most")
        expect(boxesOf(undefined)).toBe("—")
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
        expect(draft.values["boxes.minInterval"]).toBe("60")
        expect(draft.values["kinds.bomb"]).toBe("1")
        expect(draft.values["shield.maxPerBox"]).toBe("20")
        expect(draft.switches).toEqual({"frozen": false, "signInRefill": true})
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a whole count, a duration in seconds and a set frozen", () => {
        const draft = draftOf(rush())
        draft.values["spread.maxPerBox"] = "6.4"
        draft.values["welcome.boxAfter"] = "30"
        draft.switches.frozen = true

        const sent = ruleSetOf(draft).rules

        expect(sent?.spread?.maxPerBox).toBe(6)
        expect(Number(sent?.welcome?.boxAfter?.seconds)).toBe(30)
        expect(sent?.frozen).toBe(true)
        expect(offersOf(ruleSetOf(draft))).toBe("Frozen")
    })

    it("copies default as a set with no name that holds nothing, so it plays as default", () => {
        const draft = copyOf(played())

        expect(complete(draft)).toBe(false)
        expect(complete({...draft, name: "rush"})).toBe(true)
        const sent = ruleSetOf({...draft, name: "rush"})
        expect(sent.rules?.equals(new Rules())).toBe(true)
        expect(playedOf(sent, played()).rules?.boxes?.maxChargesPerHour).toBe(12)
    })

    it("copies another set with what it holds", () => {
        const quick = new RuleSet({name: "quick", rules: Rules.fromJson({boxes: {minInterval: "60s"}, frozen: true})})

        const draft = copyOf(quick)

        expect(draft.name).toBe("")
        expect(draft.values["boxes.minInterval"]).toBe("60")
        expect(draft.values["boxes.maxInterval"]).toBe("")
        expect(draft.switches.frozen).toBe(true)
        expect(draft.switches.signInRefill).toBeUndefined()
    })

    it("sends only the values a set holds, zeros and noes included", () => {
        const draft = draftOf(new RuleSet({name: "quiet"}))
        draft.values["kinds.bomb"] = "0"
        draft.switches.signInRefill = false

        expect(ruleSetOf(draft).rules?.toJson()).toEqual({kinds: {bomb: 0}, signInRefill: false})
    })

    it("plays default's values where a set holds none", () => {
        const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({boxes: {maxChargesPerHour: 6}, kinds: {bomb: 0}})})

        const rules = playedOf(quiet, played()).rules

        expect(rules?.boxes?.maxChargesPerHour).toBe(6)
        expect(Number(rules?.boxes?.minInterval?.seconds)).toBe(240)
        expect(rules?.kinds?.bomb).toBe(0)
        expect(rules?.kinds?.refill).toBe(5)
        expect(rules?.signInRefill).toBe(true)
        expect(playedOf(played(), played()).equals(played())).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name; default needs every value", () => {
        const draft = draftOf(rush())

        expect(complete(draft)).toBe(true)
        expect(complete({...draft, values: {...draft.values, "offerTtl": ""}})).toBe(true)
        expect(complete({...draftOf(played()), values: {...draftOf(played()).values, "offerTtl": ""}})).toBe(false)
        expect(complete({...draft, values: {...draft.values, "kinds.bomb": "a lot"}})).toBe(false)
        expect(complete({...draft, name: " "})).toBe(false)
    })

    it("says what changes, in the form's order, durations in the unit they read best in", () => {
        const before = draftOf(rush())
        const after = {
            ...before,
            values: {...before.values, "boxes.minInterval": "120", "offerTtl": "", "kinds.refill": "5.0"},
            switches: {...before.switches, "frozen": true, "signInRefill": undefined},
        }

        expect(changesOf(before, after)).toEqual([
            {label: "Frozen", before: "no", after: "yes"},
            {label: "Boxes · Shortest box wait (s)", before: "1 min", after: "2 min"},
            {label: "Boxes · Box flight (s)", before: "15 s", after: "default"},
            {label: "Refill at first sign-in", before: "yes", after: "default"},
        ])
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin.bonuses, {...draftOf(rush()), name: " rush "})

        const [asked] = admin.askedFor("saveBonusRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush())).toBe(true)
    })
})
