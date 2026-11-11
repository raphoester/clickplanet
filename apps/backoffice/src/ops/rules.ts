import {JsonObject} from "@bufbuild/protobuf"
import {PlanetAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {ClickRules, Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {overlay} from "./overlay.ts"
import {Field, jsonOf, numbersTyped, put, sameNumber, shownOf, typedOf, valueAt, yesOf} from "./ruleValues.ts"

export const DEFAULT_SET = "default"

export function tempoOf(set: RuleSet): string {
    return `${set.rules?.clicks?.tempo ?? 0}×`
}

// A frozen set takes no play: no click, bomb, shield or refill, and no box or quiz is caught or answered.
export function playOf(set: RuleSet): string {
    return set.rules?.frozen ? "Frozen" : "Open"
}

export function clicksOf(clicks: ClickRules | undefined): string {
    if (clicks === undefined) return "—"

    return `${clicks.burst} in hand, ${clicks.perSecond} a second`
}

// The values of the set's JSON, by their path. The boxes and quizzes are the offers' (bonusRules.ts): planet keeps
// only what a hand holds and what a charge does when it is spent.
export const GROUPS: {title: string, fields: Field[]}[] = [
    {title: "Clicks", fields: [
        {path: "clicks.tempo", label: "Refill, times as fast", kind: "number"},
        {path: "clicks.perSecond", label: "Clicks a second", kind: "number"},
        {path: "clicks.burst", label: "Clicks in hand", kind: "whole"},
        {path: "clicks.newAccountClicks", label: "New account, clicks in hand", kind: "number"},
        {path: "clicks.fastStart", label: "New account, signed-in pace for (s)", kind: "seconds"},
        {path: "clicks.linkedMultiplier", label: "Signed in, times as fast", kind: "number"},
        {path: "clicks.scopeMultiplier", label: "Address, in accounts", kind: "number"},
        {path: "clicks.v6ScopeMultiplier", label: "IPv6 address, in accounts", kind: "number"},
        {path: "clicks.guestScopeMultiplier", label: "Guests of an address, in accounts", kind: "number"},
    ]},
    {title: "Charges", fields: [
        {path: "charges.spread.clicks", label: "Spread clicks held", kind: "whole"},
        {path: "charges.bomb.rings", label: "Bomb rings", kind: "number"},
        {path: "charges.enclose.maxTiles", label: "Enclose tiles, at most", kind: "whole"},
        {path: "charges.enclose.held", label: "Enclosures held", kind: "whole"},
        {path: "charges.shield.held", label: "Shields held", kind: "whole"},
        {path: "charges.shield.perTile", label: "Shields on a tile, at most", kind: "whole"},
    ]},
    {title: "Arena", fields: [
        {path: "arena.freeze", label: "No play after the fresh map and before the end (s)", kind: "seconds"},
        {path: "seed.capitals.shields", label: "Capital seed: shields on each capital", kind: "whole"},
    ]},
]

const FIELDS = GROUPS.flatMap(group => group.fields)

// A toll step as the form edits it: the share in percent.
export type TollStep = {share: string, slowdown: string}

// A set as the form edits it: every value as typed. Outside default, "" and undefined are default's value.
export type Draft = {
    name: string, values: Record<string, string>, frozen?: boolean, allegianceLocked?: boolean, arena?: boolean,
    enclosuresUnlimited?: boolean, toll?: TollStep[], reach?: boolean, knockOut?: boolean, fixedShields?: boolean,
    bombsOff?: boolean
}

export function draftOf(set: RuleSet): Draft {
    const rules = set.rules ?? new Rules()
    const json = rules.toJson() as JsonObject

    return {
        name: set.name,
        values: Object.fromEntries(FIELDS.map(field => [field.path, typedOf(valueAt(json, field.path))])),
        enclosuresUnlimited: rules.charges?.enclose?.unlimited,
        bombsOff: rules.charges?.bomb?.off,
        frozen: rules.frozen,
        allegianceLocked: rules.allegianceLocked,
        arena: rules.arena?.on,
        reach: rules.reach,
        knockOut: rules.capitals?.knockOut,
        fixedShields: rules.capitals?.fixedShields,
        toll: rules.toll?.steps.map(step => ({
            share: String(Number((step.share * 100).toPrecision(12))), slowdown: String(step.slowdown),
        })),
    }
}

// The rules in force change only through the schedule: a new set starts as a copy of another, with no name. A copy of
// default holds nothing of its own, so it follows default.
export function copyOf(set: RuleSet): Draft {
    return set.name === DEFAULT_SET ? {name: "", values: {}} : {...draftOf(set), name: ""}
}

// Every value the set plays: its own, and default's for the rest. The toll is one value.
export function playedOf(set: RuleSet, defaults: RuleSet | undefined): RuleSet {
    if (defaults === undefined || set.name === DEFAULT_SET) return set
    const base = defaults.rules?.toJson() ?? {}
    return new RuleSet({name: set.name, rules: Rules.fromJson(overlay(base, set.rules?.toJson() ?? {}))})
}

// A name typed, and every value a number; default holds every value. The server holds the rules a set must keep.
export function complete(draft: Draft): boolean {
    const steps = (draft.toll ?? []).flatMap(step => [step.share.trim(), step.slowdown.trim()])
    return draft.name.trim() !== ""
        && numbersTyped(FIELDS, draft.values, draft.name.trim() === DEFAULT_SET)
        && steps.every(value => value !== "" && Number.isFinite(Number(value)))
}

export const SWITCHES = [
    {key: "frozen", label: "Frozen"},
    {key: "allegianceLocked", label: "Flags locked"},
    {key: "arena", label: "Arena"},
    {key: "reach", label: "Reach"},
    {key: "knockOut", label: "Knock-out"},
    {key: "fixedShields", label: "No shield on a capital"},
    {key: "enclosuresUnlimited", label: "Enclosures unlimited"},
    {key: "bombsOff", label: "Bombs off"},
] as const

// Every value the draft holds otherwise than `before`, the set it started from.
export function changesOf(before: Draft, after: Draft): Change[] {
    const changes: Change[] = []
    for (const {key, label} of SWITCHES) {
        if (before[key] !== after[key]) changes.push({label, before: yesOf(before[key]), after: yesOf(after[key])})
    }
    for (const group of GROUPS) {
        for (const field of group.fields) {
            if (sameNumber(before.values[field.path], after.values[field.path])) continue
            changes.push({
                label: `${group.title} · ${field.label}`,
                before: shownOf(field, before.values[field.path]) || "default",
                after: shownOf(field, after.values[field.path]) || "default",
            })
        }
    }
    if (tollOf(before.toll) !== tollOf(after.toll)) changes.push({label: "Toll", before: tollOf(before.toll), after: tollOf(after.toll)})
    return changes
}

export function tollOf(steps: TollStep[] | undefined): string {
    if (steps === undefined) return "default"
    return steps.length === 0 ? "none" : steps.map(step => `${step.share}% ×${step.slowdown}`).join(", ")
}

// The values the set holds, and none of default's.
export function ruleSetOf(draft: Draft): RuleSet {
    const json: JsonObject = {}
    const switches = {
        "frozen": draft.frozen, "allegianceLocked": draft.allegianceLocked, "arena.on": draft.arena, "reach": draft.reach,
        "capitals.knockOut": draft.knockOut, "capitals.fixedShields": draft.fixedShields,
        "charges.enclose.unlimited": draft.enclosuresUnlimited, "charges.bomb.off": draft.bombsOff,
    }
    for (const [path, on] of Object.entries(switches)) {
        if (on !== undefined) put(json, path, on)
    }
    for (const field of FIELDS) {
        const typed = (draft.values[field.path] ?? "").trim()
        if (typed !== "") put(json, field.path, jsonOf(Number(typed), field))
    }
    if (draft.toll !== undefined) {
        json.toll = {steps: draft.toll.map(step => ({share: Number(step.share) / 100, slowdown: Number(step.slowdown)}))}
    }

    return new RuleSet({name: draft.name.trim(), rules: Rules.fromJson(json)})
}

export async function saveRuleSet(planet: PlanetAdmin, draft: Draft): Promise<void> {
    await planet.saveRuleSet({set: ruleSetOf(draft)})
}
