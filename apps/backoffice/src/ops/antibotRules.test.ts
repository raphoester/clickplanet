import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/antibot/v1/rules_pb.ts"
import {bansOf, crowdOf, draftOf, flawOf, newDraft, offOf, saveRuleSet} from "./antibotRules.ts"

const rush = new RuleSet({
    name: "rush",
    rules: Rules.fromJson({bans: {enforce: false}, cohort: {certainMembers: 50}, metronome: {shape: {maxSkew: 0.25}}}),
})

describe("the antibot rules", () => {
    it("says whether a set bans and how big a crowd is", () => {
        expect(bansOf(rush)).toBe("Not enforced")
        expect(crowdOf(rush)).toBe("50 scopes")
        expect(bansOf(new FakeAdmin().antibotRules.sets[0])).toBe("Enforced")
    })

    it("names the watchdogs a set turns off", () => {
        expect(offOf(new FakeAdmin().antibotRules.sets[0])).toBe("defender")
        expect(offOf(rush)).toBe("retaker, sequencer, metronome, defender, catcher, cohort, scraper, churner, hopper, rotator")
    })

    it("edits a set as JSON with every value written, an unset bound left out", () => {
        const draft = draftOf(rush)
        const written = JSON.parse(draft.text) as {cohort: Record<string, unknown>, metronome: {shape: Record<string, unknown>}}

        expect(draft.name).toBe("rush")
        expect(written.cohort.minMembers).toBe(0)
        expect(written.metronome.shape.maxSkew).toBe(0.25)
        expect(written.metronome.shape).not.toHaveProperty("certainSkew")
    })

    it("starts a new set as a copy of default", () => {
        const rules = new FakeAdmin().antibotRules

        const draft = newDraft(rules)

        expect(draft.name).toBe("")
        expect(Rules.fromJsonString(draft.text).equals(rules.sets[0].rules)).toBe(true)
    })

    it("says why a text is no set, and nothing when it is one", () => {
        expect(flawOf(draftOf(rush).text)).toBeUndefined()
        expect(flawOf("{\"cohort\": {\"crowd\": 50}}")).toContain("crowd")
        expect(flawOf("{")).toBeDefined()
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin, {...draftOf(rush), name: " rush "})

        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush)).toBe(true)
    })
})
