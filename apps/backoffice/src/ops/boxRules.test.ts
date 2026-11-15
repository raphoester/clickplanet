import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/boxes/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/boxes/v1/rules_pb.ts"
import {
    boxesOf, changesOf, complete, copyOf, draftOf, FROZEN, GROUPS, offersOf, playedOf, ruleSetOf, saveRuleSet,
} from "./boxRules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const played = () => new FakeAdmin().boxRules.sets[0]

// Boxes every minute or two, thirty an hour.
const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.rules!.minInterval = minutes(1)
    set.rules!.maxInterval = minutes(2)
    set.rules!.maxBoxesPerHour = 30
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
        expect(boxesOf(played().rules)).toBe("Every 4 min to 8 min, 12 an hour at most")
        expect(boxesOf(rush().rules)).toBe("Every 1 min to 2 min, 30 an hour at most")
        expect(boxesOf(new Rules({
            minInterval: new Duration({seconds: 90n}), maxInterval: new Duration({seconds: 90n}), maxBoxesPerHour: 30,
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
        expect(draft.values["minInterval"]).toBe("60")
        expect(draft.values["maxBoxesPerHour"]).toBe("30")
        expect(draft.values["welcome.boxAfter"]).toBe("60")
        expect(draft.switches).toEqual({"frozen": false})
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a whole count, a duration in seconds and a set frozen", () => {
        const draft = draftOf(rush())
        draft.values["maxBoxesPerHour"] = "6.4"
        draft.values["welcome.boxAfter"] = "30"
        draft.switches.frozen = true

        const sent = ruleSetOf(draft).rules

        expect(sent?.maxBoxesPerHour).toBe(6)
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
        expect(playedOf(sent, played()).rules?.maxBoxesPerHour).toBe(12)
    })

    it("plays default's values where a set holds none", () => {
        const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({maxBoxesPerHour: 6})})

        const rules = playedOf(quiet, played()).rules

        expect(rules?.maxBoxesPerHour).toBe(6)
        expect(Number(rules?.minInterval?.seconds)).toBe(240)
        expect(playedOf(played(), played()).equals(played())).toBe(true)
    })

    it("says what changes, in the form's order, durations in the unit they read best in", () => {
        const before = draftOf(rush())
        const after = {
            ...before,
            values: {...before.values, "minInterval": "120", "offerTtl": ""},
            switches: {...before.switches, "frozen": true},
        }

        expect(changesOf(before, after)).toEqual([
            {label: "Frozen", before: "no", after: "yes"},
            {label: "Boxes · Shortest box wait (s)", before: "1 min", after: "2 min"},
            {label: "Boxes · Box flight (s)", before: "15 s", after: "default"},
        ])
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin.boxes, {...draftOf(rush()), name: " rush "})

        const [asked] = admin.askedFor("saveBoxRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush())).toBe(true)
    })
})
