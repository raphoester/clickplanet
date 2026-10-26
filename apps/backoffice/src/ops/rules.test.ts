import {Duration, Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {
    BoxPace,
    GetRulesResponse,
    OverrideRulesRequest,
    RuleBaseline,
    RuleOverride,
    RuleSet,
} from "../gen/grpc/planet/v1/admin_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {
    boxesOf,
    complete,
    draftOf,
    NEW_DRAFT,
    overrideMinutesValid,
    overrideRules,
    playingAt,
    ruleSetOf,
    saveRuleSet,
    tempoOf,
} from "./rules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const rush = new RuleSet({
    name: "rush",
    tempo: 3,
    boxes: new BoxPace({minInterval: minutes(1), maxInterval: minutes(2), maxChargesPerHour: 30}),
})

describe("the rules", () => {
    it("says the refill against the config's", () => {
        expect(tempoOf(new RuleSet({tempo: 1}))).toBe("As configured")
        expect(tempoOf(rush)).toBe("3× as fast")
    })

    it("says how often a box comes and the hourly cap", () => {
        expect(boxesOf(rush.boxes)).toBe("Every 1 min to 2 min, 30 an hour at most")
        expect(boxesOf(new BoxPace({minInterval: minutes(4), maxInterval: minutes(4), maxChargesPerHour: 12})))
            .toBe("Every 4 min, 12 an hour at most")
        expect(boxesOf(new BoxPace({
            minInterval: new Duration({seconds: 90n}), maxInterval: minutes(2), maxChargesPerHour: 30,
        }))).toBe("Every 90 s to 2 min, 30 an hour at most")
        expect(boxesOf(undefined)).toBe("—")
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush)

        expect(draft).toEqual({name: "rush", tempo: "3", minSeconds: "60", maxSeconds: "120", cap: "30"})
        expect(ruleSetOf(draft).equals(rush)).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name", () => {
        expect(complete(NEW_DRAFT)).toBe(false)
        expect(complete({...NEW_DRAFT, name: "rush"})).toBe(true)
        expect(complete({...NEW_DRAFT, name: "rush", tempo: ""})).toBe(false)
        expect(complete({...NEW_DRAFT, name: "rush", cap: "a lot"})).toBe(false)
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin, draftOf(rush))

        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush)).toBe(true)
    })
})

describe("what the game plays", () => {
    const at = new Date("2026-10-31T21:00:00Z")
    const ts = (offsetMinutes: number) => Timestamp.fromDate(new Date(at.getTime() + offsetMinutes * 60_000))
    const finale = new RuleBaseline({requestId: "season-0:finale", ruleSet: "rush", appliedAt: ts(0)})
    const days = new RuleBaseline({requestId: "season-0:days", ruleSet: "default", appliedAt: ts(-60)})

    it("is the default before any baseline", () => {
        expect(playingAt(new GetRulesResponse(), at)).toEqual({layer: "default", set: "default"})
    })

    it("is the newest baseline while no override runs", () => {
        const rules = new GetRulesResponse({
            baselines: [finale, days],
            override: new RuleOverride({ruleSet: "default", startsAt: ts(-30), endsAt: ts(-10)}),
        })

        expect(playingAt(rules, at)).toEqual({
            layer: "baseline", set: "rush", request: "season-0:finale", since: finale.appliedAt?.toDate(),
        })
    })

    it("is the override from its start until its end", () => {
        const rules = new GetRulesResponse({
            baselines: [finale],
            override: new RuleOverride({ruleSet: "default", startsAt: ts(0), endsAt: ts(30)}),
        })

        expect(playingAt(rules, at)).toEqual({layer: "override", set: "default", until: ts(30).toDate()})
        expect(playingAt(rules, ts(30).toDate()).layer).toBe("baseline")
    })
})

describe("an override", () => {
    it("runs a whole number of minutes, a day at most", () => {
        expect(["30", "1440"].map(overrideMinutesValid)).toEqual([true, true])
        expect(["", "0", "1.5", "1441", "-5"].map(overrideMinutesValid)).toEqual([false, false, false, false, false])
    })

    it("asks for an end that many minutes from now", async () => {
        const admin = new FakeAdmin()
        const now = new Date("2026-10-31T21:00:00Z")

        await overrideRules(admin, "rush", 30, now)

        const [asked] = admin.askedFor("overrideRules") as OverrideRulesRequest[]
        expect(asked.ruleSet).toBe("rush")
        expect(asked.until?.toDate()).toEqual(new Date("2026-10-31T21:30:00Z"))
    })
})
