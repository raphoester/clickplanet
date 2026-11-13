import {BonusesAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {BoxPace, Rules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import * as ruleBook from "./ruleBook.ts"
import {Book, Draft, Group} from "./ruleBook.ts"
import {paceOf, Switch} from "./ruleValues.ts"

// The offers' rule sets (bonuses.v1): when a box comes, what a box or a right answer may be and how much it grants.
// What a hand holds and what a charge does are planet's (the Game rules' Charges); when a quiz comes is the quizzes'.

export type {Draft}

// Offered nothing, and every claim refused, while it plays.
export const FROZEN: Switch = {path: "frozen", label: "Frozen: no box is offered or caught", short: "Frozen"}

export const GROUPS: Group[] = [
    {title: "Boxes", switches: [], fields: [
        {path: "boxes.minInterval", label: "Shortest box wait (s)", kind: "seconds"},
        {path: "boxes.maxInterval", label: "Longest box wait (s)", kind: "seconds"},
        {path: "boxes.maxChargesPerHour", label: "Charges an hour, at most", kind: "whole"},
        {path: "offerTtl", label: "Box flight (s)", kind: "seconds"},
        {path: "missRetry", label: "Next box after a miss (s)", kind: "seconds"},
        {path: "activeWithin", label: "Last click within (s)", kind: "seconds"},
    ]},
    {title: "What a box or a right answer is", switches: [], fields: [
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
]

export const BOOK: Book<RuleSet> = {
    frozen: FROZEN,
    groups: GROUPS,
    setOf: (name, json) => new RuleSet({name, rules: Rules.fromJson(json)}),
}

export function draftOf(set: RuleSet): Draft {
    return ruleBook.draftOf(BOOK, set)
}

export function copyOf(set: RuleSet): Draft {
    return ruleBook.copyOf(BOOK, set)
}

export function playedOf(set: RuleSet, defaults: RuleSet | undefined): RuleSet {
    return ruleBook.playedOf(BOOK, set, defaults)
}

export function complete(draft: Draft): boolean {
    return ruleBook.complete(BOOK, draft)
}

export function changesOf(before: Draft, after: Draft): Change[] {
    return ruleBook.changesOf(BOOK, before, after)
}

export function ruleSetOf(draft: Draft): RuleSet {
    return ruleBook.ruleSetOf(BOOK, draft)
}

export async function saveRuleSet(bonuses: BonusesAdmin, draft: Draft): Promise<void> {
    await bonuses.saveRuleSet({set: ruleSetOf(draft)})
}

// A frozen set offers nothing, and refuses every claim.
export function offersOf(set: RuleSet): string {
    return set.rules?.frozen ? "Frozen" : "Open"
}

export function boxesOf(boxes: BoxPace | undefined): string {
    if (boxes === undefined) return "—"

    return `Every ${paceOf(boxes.minInterval, boxes.maxInterval)}, ${boxes.maxChargesPerHour ?? 0} an hour at most`
}
