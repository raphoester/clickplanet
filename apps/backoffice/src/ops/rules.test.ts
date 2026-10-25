import {Duration} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {AntiBotRules, BoxPace, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {boxesOf, complete, crowdOf, draftOf, NEW_DRAFT, ruleSetOf, saveRuleSet, tempoOf} from "./rules.ts"

const minutes = (n: number) => new Duration({seconds: BigInt(n * 60)})

const rush = new RuleSet({
    name: "rush",
    tempo: 3,
    boxes: new BoxPace({minInterval: minutes(1), maxInterval: minutes(2), maxChargesPerHour: 30}),
    antiBot: new AntiBotRules({crowd: 50}),
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

    it("says from how many networks of one range a crowd is banned", () => {
        expect(crowdOf(rush)).toBe("A crowd from 50 networks")
        expect(crowdOf(new RuleSet())).toBe("—")
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush)

        expect(draft).toEqual({name: "rush", tempo: "3", minSeconds: "60", maxSeconds: "120", cap: "30", crowd: "50"})
        expect(ruleSetOf(draft).equals(rush)).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name", () => {
        expect(complete(NEW_DRAFT)).toBe(false)
        expect(complete({...NEW_DRAFT, name: "rush"})).toBe(true)
        expect(complete({...NEW_DRAFT, name: "rush", tempo: ""})).toBe(false)
        expect(complete({...NEW_DRAFT, name: "rush", crowd: "a lot"})).toBe(false)
    })

    it("saves a set as the form holds it", async () => {
        const admin = new FakeAdmin()

        await saveRuleSet(admin, draftOf(rush))

        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.equals(rush)).toBe(true)
    })
})
