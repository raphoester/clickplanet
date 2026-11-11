import {Duration, JsonObject} from "@bufbuild/protobuf"
import {BonusesAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {BoxPace, QuizRules, Rules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import {overlay} from "./overlay.ts"
import {DEFAULT_SET} from "./rules.ts"
import {Field, jsonOf, numbersTyped, put, sameNumber, shownOf, Switch, typedOf, valueAt, yesOf} from "./ruleValues.ts"

// The offers' rule sets (bonuses.v1): when a box and a quiz come, what each may be and how much it grants. What a hand
// holds and what a charge does are planet's (the Game rules' Charges).

// Offered nothing, and every claim and answer refused, while it plays.
export const FROZEN: Switch = {path: "frozen", label: "Frozen: no box or quiz is offered, caught or answered", short: "Frozen"}

// Each group of the form: its switches first, then a field per value.
export const GROUPS: {title: string, switches: Switch[], fields: Field[]}[] = [
    {title: "Boxes", switches: [], fields: [
        {path: "boxes.minInterval", label: "Shortest box wait (s)", kind: "seconds"},
        {path: "boxes.maxInterval", label: "Longest box wait (s)", kind: "seconds"},
        {path: "boxes.maxChargesPerHour", label: "Charges an hour, at most", kind: "whole"},
        {path: "offerTtl", label: "Box flight (s)", kind: "seconds"},
        {path: "missRetry", label: "Next box after a miss (s)", kind: "seconds"},
        {path: "activeWithin", label: "Last click within (s)", kind: "seconds"},
    ]},
    {title: "What a box or a quiz is", switches: [], fields: [
        {path: "kinds.refill", label: "Refill, weight", kind: "number"},
        {path: "kinds.spreadClicks", label: "Spread, weight", kind: "number"},
        {path: "kinds.encloseClicks", label: "Enclose, weight", kind: "number"},
        {path: "kinds.bomb", label: "Bomb, weight", kind: "number"},
        {path: "kinds.shields", label: "Shields, weight", kind: "number"},
    ]},
    {title: "What it grants", switches: [], fields: [
        {path: "spread.maxPerBox", label: "Spread clicks a box, at most", kind: "whole"},
        {path: "enclose.maxPerBox", label: "Enclosures a box, at most", kind: "whole"},
        {path: "shield.minPerBox", label: "Shields a box, at least", kind: "whole"},
        {path: "shield.maxPerBox", label: "Shields a box, at most", kind: "whole"},
    ]},
    {
        title: "New accounts",
        switches: [{path: "signInRefill", label: "Refill at first sign-in", short: "Refill at first sign-in"}],
        fields: [
            {path: "welcome.within", label: "New for (s)", kind: "seconds"},
            {path: "welcome.boxAfter", label: "First box after (s)", kind: "seconds"},
        ],
    },
    {
        title: "Quizzes",
        switches: [{path: "quiz.enabled", label: "Quizzes on", short: "Quizzes on"}],
        fields: [
            {path: "quiz.minInterval", label: "Shortest quiz wait (s)", kind: "seconds"},
            {path: "quiz.maxInterval", label: "Longest quiz wait (s)", kind: "seconds"},
            {path: "quiz.maxChargesPerHour", label: "Quizzes won an hour, at most", kind: "whole"},
            {path: "quiz.offerTtl", label: "Banner (s)", kind: "seconds"},
            {path: "quiz.answerWindow", label: "Time to answer (s)", kind: "seconds"},
            {path: "quiz.missRetry", label: "Next quiz after a miss (s)", kind: "seconds"},
            {path: "quiz.leaderBias", label: "Lean to the leaders", kind: "number"},
        ],
    },
]

const FIELDS = GROUPS.flatMap(group => group.fields)
const SWITCHES = [FROZEN, ...GROUPS.flatMap(group => group.switches)]

// A set as the form edits it: every value as typed, every switch, by its path. Outside default, "" and undefined are
// default's value.
export type Draft = {name: string, values: Record<string, string>, switches: Record<string, boolean | undefined>}

export function draftOf(set: RuleSet): Draft {
    const json = (set.rules ?? new Rules()).toJson() as JsonObject
    return {
        name: set.name,
        values: Object.fromEntries(FIELDS.map(field => [field.path, typedOf(valueAt(json, field.path))])),
        switches: Object.fromEntries(SWITCHES.map(({path}) => {
            const on = valueAt(json, path)
            return [path, typeof on === "boolean" ? on : undefined]
        })),
    }
}

// The offers play the schedule's set: a new one starts as a copy of another, with no name. A copy of default holds
// nothing of its own, so it follows default.
export function copyOf(set: RuleSet): Draft {
    return {...draftOf(set.name === DEFAULT_SET ? new RuleSet() : set), name: ""}
}

// Every value the set plays: its own, and default's for the rest.
export function playedOf(set: RuleSet, defaults: RuleSet | undefined): RuleSet {
    if (defaults === undefined || set.name === DEFAULT_SET) return set
    const base = defaults.rules?.toJson() ?? {}
    return new RuleSet({name: set.name, rules: Rules.fromJson(overlay(base, set.rules?.toJson() ?? {}))})
}

// A name typed, and every value a number; default holds every value. The server holds the rules a set must keep.
export function complete(draft: Draft): boolean {
    return draft.name.trim() !== "" && numbersTyped(FIELDS, draft.values, draft.name.trim() === DEFAULT_SET)
}

// Every value the draft holds otherwise than `before`, the set it started from, in the form's order.
export function changesOf(before: Draft, after: Draft): Change[] {
    const changes: Change[] = []
    const switched = (each: Switch) => {
        const was = before.switches[each.path]
        const is = after.switches[each.path]
        if (was !== is) changes.push({label: each.short, before: yesOf(was), after: yesOf(is)})
    }
    switched(FROZEN)
    for (const group of GROUPS) {
        group.switches.forEach(switched)
        for (const field of group.fields) {
            if (sameNumber(before.values[field.path], after.values[field.path])) continue
            changes.push({
                label: `${group.title} · ${field.label}`,
                before: shownOf(field, before.values[field.path]) || "default",
                after: shownOf(field, after.values[field.path]) || "default",
            })
        }
    }
    return changes
}

// The values the set holds, and none of default's.
export function ruleSetOf(draft: Draft): RuleSet {
    const json: JsonObject = {}
    for (const {path} of SWITCHES) {
        const on = draft.switches[path]
        if (on !== undefined) put(json, path, on)
    }
    for (const field of FIELDS) {
        const typed = (draft.values[field.path] ?? "").trim()
        if (typed !== "") put(json, field.path, jsonOf(Number(typed), field))
    }
    return new RuleSet({name: draft.name.trim(), rules: Rules.fromJson(json)})
}

export async function saveRuleSet(bonuses: BonusesAdmin, draft: Draft): Promise<void> {
    await bonuses.saveRuleSet({set: ruleSetOf(draft)})
}

// A frozen set offers nothing, and refuses every claim and answer.
export function offersOf(set: RuleSet): string {
    return set.rules?.frozen ? "Frozen" : "Open"
}

export function boxesOf(boxes: BoxPace | undefined): string {
    if (boxes === undefined) return "—"

    return `Every ${paceOf(boxes.minInterval, boxes.maxInterval)}, ${boxes.maxChargesPerHour ?? 0} an hour at most`
}

export function quizzesOf(quiz: QuizRules | undefined): string {
    if (quiz === undefined) return "—"
    if (quiz.enabled !== true) return "Off"

    return `Every ${paceOf(quiz.minInterval, quiz.maxInterval)}, ${quiz.maxChargesPerHour ?? 0} won an hour at most`
}

function paceOf(min: Duration | undefined, max: Duration | undefined): string {
    const shortest = spanOf(min)
    const longest = spanOf(max)
    return shortest === longest ? shortest : `${shortest} to ${longest}`
}

function spanOf(duration: Duration | undefined): string {
    const seconds = Number(duration?.seconds ?? 0n)
    return seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`
}
