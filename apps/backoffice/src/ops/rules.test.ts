import {Duration} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {AntiBotRules, BoxPace, GetRulesResponse, RuleChange, RuleChangeSource, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {applyRuleSet, boxesOf, crowdOf, inForceOf, sourceOf, tempoOf} from "./rules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

describe("the rules", () => {
    it("is the newest change in force, and the default with none", () => {
        expect(inForceOf(new GetRulesResponse())).toBe("default")
        expect(inForceOf(new GetRulesResponse({
            changes: [new RuleChange({ruleSet: "rush"}), new RuleChange({ruleSet: "default"})],
        }))).toBe("rush")
    })

    it("says the refill against the config's", () => {
        expect(tempoOf(new RuleSet({tempo: 1}))).toBe("As configured")
        expect(tempoOf(new RuleSet({tempo: 3}))).toBe("3× as fast")
    })

    it("says how often a box comes and the hourly cap", () => {
        expect(boxesOf(new BoxPace({minInterval: minutes(1), maxInterval: minutes(2), maxChargesPerHour: 30})))
            .toBe("Every 1 min to 2 min, 30 an hour at most")
        expect(boxesOf(new BoxPace({minInterval: minutes(4), maxInterval: minutes(4), maxChargesPerHour: 12})))
            .toBe("Every 4 min, 12 an hour at most")
        expect(boxesOf(new BoxPace({
            minInterval: new Duration({seconds: 90n}), maxInterval: minutes(2), maxChargesPerHour: 30,
        }))).toBe("Every 90 s to 2 min, 30 an hour at most")
        expect(boxesOf(undefined)).toBe("—")
    })

    it("says from how many networks of one range a crowd is banned", () => {
        expect(crowdOf(new RuleSet({antiBot: new AntiBotRules({crowd: 50})}))).toBe("A crowd from 50 networks")
        expect(crowdOf(new RuleSet())).toBe("—")
    })

    it("names who changed them", () => {
        expect(sourceOf(RuleChangeSource.INTERNAL)).toBe("Seasons")
        expect(sourceOf(RuleChangeSource.OPERATOR)).toBe("An operator")
        expect(sourceOf(RuleChangeSource.UNSPECIFIED)).toBe("Unknown")
    })

    it("applies a set under a request of its own", async () => {
        const admin = new FakeAdmin()

        await applyRuleSet(admin, "default", "request-1")

        expect(admin.askedFor("applyRuleSet")).toEqual([{requestId: "request-1", ruleSet: "default"}])
    })
})
