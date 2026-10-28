import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/antibot/v1/rules_pb.ts"
import {
    bansOf, changesOf, copyOf, crowdOf, draftOf, flawOf, LEAVES, offOf, PARTS, rulesOf, saveRuleSet,
} from "./antibotRules.ts"

const rush = new RuleSet({
    name: "rush",
    rules: Rules.fromJson({
        bans: {enforce: false, banDurations: ["86400s", "604800s"]},
        cohort: {certainMembers: 50},
        metronome: {maxGap: "7s", shape: {maxSkew: 0.25}},
    }),
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

    it("reads the form off the proto: a part each, its switch first, and every value of antibot.v1.Rules", () => {
        expect(PARTS.map(part => part.label)).toEqual([
            "Bans", "Jury", "Retaker", "Sequencer", "Metronome", "Defender", "Catcher", "Cohort", "Scraper", "Churner",
            "Hopper", "Rotator",
        ])
        const metronome = PARTS.find(part => part.label === "Metronome")
        expect(metronome?.leaves[0]).toEqual({path: "metronome.enabled", label: "Enabled", kind: "switch", optional: false})
        expect(metronome?.groups.map(group => group.label)).toEqual(["Shape", "Clock", "Stamina"])
        expect(LEAVES.find(leaf => leaf.path === "metronome.shape.maxSkew")).toMatchObject({kind: "number", optional: true})
        expect(LEAVES.find(leaf => leaf.path === "bans.banDurations")).toMatchObject({kind: "secondsList"})
        expect(LEAVES.find(leaf => leaf.path === "cohort.certainMembers")).toMatchObject({kind: "whole", optional: false})
        expect(LEAVES).toHaveLength(94)
    })

    it("edits a set's values as typed: durations in seconds, an unset bound empty, an unset part as zeros", () => {
        const draft = draftOf(rush)
        expect(draft.name).toBe("rush")
        expect(draft.values["bans.enforce"]).toBe(false)
        expect(draft.values["bans.banDurations"]).toEqual(["86400", "604800"])
        expect(draft.values["metronome.maxGap"]).toBe("7")
        expect(draft.values["metronome.shape.maxSkew"]).toBe("0.25")
        expect(draft.values["metronome.shape.certainSkew"]).toBe("")
        expect(draft.values["metronome.maxSpread"]).toBe("")
        expect(draft.values["cohort.minMembers"]).toBe("0")
        expect(draft.values["hopper.minShare"]).toBe("0")
    })

    it("gives back the set it was read from, a part it left unset as zeros", () => {
        const draft = draftOf(rush)
        const back = rulesOf(draft)
        expect(draftOf(new RuleSet({name: "rush", rules: back})).values).toEqual(draft.values)
        expect(back.bans?.banDurations.map(duration => Number(duration.seconds))).toEqual([86400, 604800])
        expect(back.metronome?.shape?.maxSkew).toBe(0.25)
        expect(back.metronome?.shape?.certainSkew).toBeUndefined()
        expect(back.metronome?.maxSpread).toBeUndefined()
        const copy = copyOf(rush)
        expect(copy.name).toBe("")
        expect(copy.values).toEqual(draft.values)
    })

    it("lists what the form changes, durations in the unit they read best in", () => {
        const before = draftOf(rush)
        const after = {
            ...before,
            values: {
                ...before.values, "bans.enforce": true, "metronome.maxGap": "300", "cohort.certainMembers": "50.0",
                "bans.banDurations": ["86400", "604800", "94608000"],
            },
        }
        expect(changesOf(before, after)).toEqual([
            {label: "Bans · Enforce", before: "off", after: "on"},
            {label: "Bans · Ban durations", before: "1 d, 7 d", after: "1 d, 7 d, 1095 d"},
            {label: "Metronome · Max gap", before: "7 s", after: "5 min"},
        ])
    })

    it("says why a draft is no set, and nothing when it is one", () => {
        const draft = draftOf(rush)
        expect(flawOf(draft)).toBeUndefined()
        expect(flawOf({...draft, values: {...draft.values, "cohort.minMembers": ""}})).toBe("Cohort · Min members: a number is needed.")
        expect(flawOf({...draft, values: {...draft.values, "metronome.maxGap": "soon"}})).toBe("Metronome · Max gap: soon is not a number.")
        expect(flawOf({...draft, values: {...draft.values, "bans.banDurations": ["86400", ""]}})).toBe("Bans · Ban durations: a step is empty.")
        expect(flawOf({...draft, values: {...draft.values, "metronome.shape.maxSkew": ""}})).toBeUndefined()
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()
        await saveRuleSet(admin.antibot, {...draftOf(rush), name: " rush "})
        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.name).toBe("rush")
        expect(draftOf(asked.set).values).toEqual(draftOf(rush).values)
    })
})
