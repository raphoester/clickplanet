import {JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import {
    BOOK, changesOf, complete, copyOf, draftOf, GROUPS, kindsOf, playedOf, ruleSetOf, saveRuleSet,
} from "./bonusRules.ts"

const played = () => new FakeAdmin().bonusRules.sets[0]

// Bombs three times as likely, and more shields a box.
const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.rules!.kinds!.bomb = 3
    set.rules!.shield!.maxPerBox = 30
    return set
}

function leaves(json: JsonValue, path = ""): string[] {
    if (typeof json !== "object" || json === null || Array.isArray(json)) return [path]
    return Object.entries(json as JsonObject).flatMap(([key, value]) => leaves(value, path === "" ? key : `${path}.${key}`))
}

describe("what a box or a right answer holds", () => {
    it("says what a box may be drawn as, and each kind's weight", () => {
        expect(kindsOf(played().rules)).toBe("refill 5, spread 3, enclose 2, bomb 1, shields 3")
        expect(kindsOf(Rules.fromJson({kinds: {bomb: 0}}))).toBe("bomb 0")
        expect(kindsOf(undefined)).toBe("—")
    })

    it("edits every value of a set, and has no frozen set: the boxes and the quizzes freeze", () => {
        const edited = new Set(GROUPS.flatMap(group => [...group.switches, ...group.fields].map(each => each.path)))

        const values = leaves(played().rules!.toJson({emitDefaultValues: true}))

        expect(values.filter(path => !edited.has(path))).toEqual([])
        expect([...edited].filter(path => !values.includes(path))).toEqual([])
        expect(BOOK.frozen).toBeUndefined()
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush())

        expect(draft.name).toBe("rush")
        expect(draft.values["kinds.bomb"]).toBe("3")
        expect(draft.values["shield.maxPerBox"]).toBe("30")
        expect(draft.switches).toEqual({"signInRefill": true})
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a whole count", () => {
        const draft = draftOf(rush())
        draft.values["spread.maxPerBox"] = "6.4"

        expect(ruleSetOf(draft).rules?.spread?.maxPerBox).toBe(6)
    })

    it("copies default as a set with no name that holds nothing, so it plays as default", () => {
        const draft = copyOf(played())

        expect(complete(draft)).toBe(false)
        expect(complete({...draft, name: "rush"})).toBe(true)
        const sent = ruleSetOf({...draft, name: "rush"})
        expect(sent.rules?.equals(new Rules())).toBe(true)
        expect(playedOf(sent, played()).rules?.kinds?.refill).toBe(5)
    })

    it("sends only the values a set holds, zeros and noes included", () => {
        const draft = draftOf(new RuleSet({name: "quiet"}))
        draft.values["kinds.bomb"] = "0"
        draft.switches.signInRefill = false

        expect(ruleSetOf(draft).rules?.toJson()).toEqual({kinds: {bomb: 0}, signInRefill: false})
    })

    it("plays default's values where a set holds none", () => {
        const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({kinds: {bomb: 0}})})

        const rules = playedOf(quiet, played()).rules

        expect(rules?.kinds?.bomb).toBe(0)
        expect(rules?.kinds?.refill).toBe(5)
        expect(rules?.signInRefill).toBe(true)
        expect(playedOf(played(), played()).equals(played())).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name; default needs every value", () => {
        const draft = draftOf(rush())

        expect(complete(draft)).toBe(true)
        expect(complete({...draft, values: {...draft.values, "spread.maxPerBox": ""}})).toBe(true)
        expect(complete({...draftOf(played()), values: {...draftOf(played()).values, "spread.maxPerBox": ""}})).toBe(false)
        expect(complete({...draft, values: {...draft.values, "kinds.bomb": "a lot"}})).toBe(false)
        expect(complete({...draft, name: " "})).toBe(false)
    })

    it("says what changes, in the form's order", () => {
        const before = draftOf(rush())
        const after = {
            ...before,
            values: {...before.values, "kinds.refill": "5.0", "kinds.bomb": "1", "shield.maxPerBox": ""},
            switches: {...before.switches, "signInRefill": undefined},
        }

        expect(changesOf(before, after)).toEqual([
            {label: "What a box or a right answer is · Bomb, weight", before: "3", after: "1"},
            {label: "What it grants · Shields a box, at most", before: "30", after: "default"},
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
