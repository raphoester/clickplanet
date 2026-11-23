import {JsonObject, JsonValue} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {ClickRules, Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {changesOf, clicksOf, complete, copyOf, draftOf, GROUPS, playedOf, playOf, ruleSetOf, saveRuleSet, tempoOf} from "./rules.ts"

const played = () => new FakeAdmin().rules.sets[0]

const rush = () => {
    const set = played().clone()
    set.name = "rush"
    set.rules!.clicks!.tempo = 3
    set.rules!.charges!.bomb!.rings = 6
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

    it("edits every value of a set", () => {
        const edited = new Set([...GROUPS.flatMap(group => group.fields.map(field => field.path)), "charges.enclose.unlimited", "charges.spread.unlimited", "toll.steps", "frozen", "allegianceLocked", "arena.on", "reach", "capitals.knockOut", "capitals.fixedShields", "capitals.cutOff", "charges.bomb.off", "charges.enclose.off", "charges.spread.off", "charges.rally.off", "arena.board"])

        const values = leaves(played().rules!.toJson({emitDefaultValues: true}))

        expect(values.filter(path => !edited.has(path))).toEqual([])
    })

    it("holds a hand's charges, and nothing of the boxes or quizzes", () => {
        const paths = GROUPS.flatMap(group => group.fields.map(field => field.path))

        expect(paths.filter(path => path.startsWith("charges."))).toEqual([
            "charges.refill.held", "charges.spread.clicks", "charges.spread.rings", "charges.bomb.held", "charges.bomb.rings",
            "charges.enclose.maxTiles", "charges.enclose.held", "charges.shield.held", "charges.shield.perTile",
            "charges.virus.held", "charges.virus.infections", "charges.virus.every",
            "charges.mine.held", "charges.mine.rings",
            "charges.rally.held", "charges.rally.rings", "charges.rally.lasts", "charges.rally.cost",
            "charges.trap.held", "charges.trap.clicks",
        ])
        expect(paths.filter(path => /box|quiz|kinds|welcome|offer/i.test(path))).toEqual([])
    })

    it("keeps the board an arena plays through an edit, even one the page does not carry, and says a board changed", () => {
        const rules = Rules.fromJson({...rush().rules!.toJson() as JsonObject, arena: {on: true, board: "siege-1"}})
        const set = new RuleSet({name: "siege", rules})

        const draft = draftOf(set)
        expect(draft.board).toBe("siege-1")
        expect(ruleSetOf(draft).equals(set)).toBe(true)

        expect(changesOf(draft, {...draft, board: "earth-1"})).toEqual([
            {label: "Arena · Board its fresh map plays", before: "siege-1", after: "earth-1"},
        ])
        expect(changesOf(draft, {...draft, board: undefined})).toEqual([
            {label: "Arena · Board its fresh map plays", before: "siege-1", after: "default"},
        ])
        expect(ruleSetOf({...draft, board: ""}).rules?.arena?.board).toBe("")
        expect(ruleSetOf({...draft, board: undefined}).rules?.arena?.board).toBeUndefined()
    })

    it("holds no board in default for the world's", () => {
        const draft = draftOf(played())

        expect(draft.board).toBeUndefined()
        expect(changesOf(draft, {...draft, board: ""})).toEqual([])
        expect(changesOf(draft, {...draft, board: "earth-1"})).toEqual([
            {label: "Arena · Board its fresh map plays", before: "the world's", after: "earth-1"},
        ])
    })

    it("edits a set as it is, and sends back what was edited", () => {
        const draft = draftOf(rush())

        expect(draft.name).toBe("rush")
        expect(draft.values["clicks.tempo"]).toBe("3")
        expect(draft.values["charges.bomb.rings"]).toBe("6")
        expect(draft.values["clicks.perSecond"]).toBe("0.2")
        expect(draft.values["charges.shield.held"]).toBe("30")
        expect(draft.values["charges.refill.held"]).toBe("1")
        expect(draft.values["charges.bomb.held"]).toBe("1")
        expect(draft.values["charges.virus.held"]).toBe("1")
        expect(draft.bombsOff).toBe(false)
        expect(draft.toll).toEqual([{share: "10", slowdown: "1.5"}, {share: "20", slowdown: "2.5"}, {share: "30", slowdown: "4"}])
        expect(ruleSetOf(draft).equals(rush())).toBe(true)
    })

    it("sends a toll step added, bombs switched off and a whole count", () => {
        const draft = draftOf(rush())
        draft.toll = [...draft.toll ?? [], {share: "45", slowdown: "6"}]
        draft.bombsOff = true
        draft.values["charges.shield.perTile"] = "12.4"

        const sent = ruleSetOf(draft).rules

        expect(sent?.toll?.steps.at(-1)?.share).toBeCloseTo(0.45)
        expect(sent?.toll?.steps.at(-1)?.slowdown).toBe(6)
        expect(sent?.charges?.bomb?.off).toBe(true)
        expect(sent?.charges?.shield?.perTile).toBe(12)
    })

    it("sends bombs switched off, and says it changed", () => {
        const before = draftOf(played())
        const after = {...before, bombsOff: true}

        expect(before.bombsOff).toBe(false)
        expect(ruleSetOf(after).rules?.charges?.bomb?.off).toBe(true)
        expect(changesOf(before, after)).toEqual([{label: "Bombs off", before: "no", after: "yes"}])
    })

    it("sends enclose switched off, and says it changed", () => {
        const before = draftOf(played())
        const after = {...before, enclosuresOff: true}

        expect(before.enclosuresOff).toBe(false)
        expect(ruleSetOf(after).rules?.charges?.enclose?.off).toBe(true)
        expect(changesOf(before, after)).toEqual([{label: "Enclose off", before: "no", after: "yes"}])
    })

    it("sends spread switched off, and says it changed", () => {
        const before = draftOf(played())
        const after = {...before, spreadOff: true}

        expect(before.spreadOff).toBe(false)
        expect(ruleSetOf(after).rules?.charges?.spread?.off).toBe(true)
        expect(changesOf(before, after)).toEqual([{label: "Spread off", before: "no", after: "yes"}])
    })

    it("sends rallies switched off, and says it changed", () => {
        const before = draftOf(played())
        const after = {...before, ralliesOff: true}

        expect(before.ralliesOff).toBe(false)
        expect(ruleSetOf(after).rules?.charges?.rally?.off).toBe(true)
        expect(changesOf(before, after)).toEqual([{label: "Rally off", before: "no", after: "yes"}])
    })

    it("sends unlimited enclosures switched on, and says it changed", () => {
        const before = draftOf(played())
        const after = {...before, enclosuresUnlimited: true}

        expect(before.enclosuresUnlimited).toBe(false)
        expect(ruleSetOf(after).rules?.charges?.enclose?.unlimited).toBe(true)
        expect(changesOf(before, after)).toEqual([{label: "Enclosures unlimited", before: "no", after: "yes"}])
    })

    it("sends unlimited spreads switched on and the spread's rings, and says they changed", () => {
        const before = draftOf(played())
        const after = {...before, spreadUnlimited: true, values: {...before.values, "charges.spread.rings": "2"}}

        expect(before.spreadUnlimited).toBe(false)
        expect(before.values["charges.spread.rings"]).toBe("1")
        expect(ruleSetOf(after).rules?.charges?.spread?.unlimited).toBe(true)
        expect(ruleSetOf(after).rules?.charges?.spread?.rings).toBe(2)
        expect(changesOf(before, after)).toEqual([
            {label: "Spread unlimited", before: "no", after: "yes"},
            {label: "Charges · Spread rings, 1 to 3", before: "1", after: "2"},
        ])
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
        expect(draft.bombsOff).toBeUndefined()
    })

    it("sends only the values a set holds, zeros and an empty toll included", () => {
        const draft = draftOf(new RuleSet({name: "quiet"}))
        draft.values["clicks.newAccountClicks"] = "0"
        draft.bombsOff = false
        draft.toll = []

        expect(ruleSetOf(draft).rules?.toJson()).toEqual({clicks: {newAccountClicks: 0}, toll: {}, charges: {bomb: {off: false}}})
    })

    it("plays default's values where a set holds none, and a toll it holds whole", () => {
        const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({clicks: {tempo: 3}, charges: {bomb: {off: true}}, toll: {}})})

        const rules = playedOf(quiet, played()).rules

        expect(rules?.clicks?.tempo).toBe(3)
        expect(rules?.clicks?.burst).toBe(60)
        expect(rules?.charges?.bomb?.off).toBe(true)
        expect(rules?.charges?.bomb?.rings).toBe(4)
        expect(rules?.toll?.steps).toEqual([])
        expect(playedOf(played(), played()).equals(played())).toBe(true)
    })

    it("sends nothing until every value is a number and the set has a name; default needs every value", () => {
        const draft = draftOf(rush())

        expect(complete(draft)).toBe(true)
        expect(complete({...draft, values: {...draft.values, "clicks.tempo": ""}})).toBe(true)
        expect(complete({...draftOf(played()), values: {...draftOf(played()).values, "clicks.tempo": ""}})).toBe(false)
        expect(complete({...draft, values: {...draft.values, "charges.bomb.rings": "a lot"}})).toBe(false)
        expect(complete({...draft, toll: [{share: "10", slowdown: ""}]})).toBe(false)
    })

    it("says a value that follows default", () => {
        const before = draftOf(rush())
        const after = {...before, values: {...before.values, "clicks.tempo": ""}, bombsOff: undefined, toll: undefined}

        expect(changesOf(before, after)).toEqual([
            {label: "Bombs off", before: "no", after: "default"},
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
