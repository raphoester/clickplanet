import {BoxesAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {RuleSet} from "../gen/grpc/boxes/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/boxes/v1/rules_pb.ts"
import * as ruleBook from "./ruleBook.ts"
import {Book, Draft, Group} from "./ruleBook.ts"
import {paceOf, Switch} from "./ruleValues.ts"

// The boxes' rule sets (boxes.v1): when a box comes, how long it flies, and a newcomer's first box. What a box holds is
// the bonuses module's (bonusRules.ts).

export type {Draft}

// No box offered, and every claim refused, while it plays.
export const FROZEN: Switch = {path: "frozen", label: "Frozen: no box is offered or caught", short: "Frozen"}

export const GROUPS: Group[] = [
    {title: "Boxes", switches: [], fields: [
        {path: "minInterval", label: "Shortest box wait (s)", kind: "seconds"},
        {path: "maxInterval", label: "Longest box wait (s)", kind: "seconds"},
        {path: "maxBoxesPerHour", label: "Boxes an hour, at most", kind: "whole"},
        {path: "offerTtl", label: "Box flight (s)", kind: "seconds"},
        {path: "missRetry", label: "Next box after a miss (s)", kind: "seconds"},
        {path: "activeWithin", label: "Last click within (s)", kind: "seconds"},
    ]},
    {title: "New accounts", switches: [], fields: [
        {path: "welcome.within", label: "New for (s)", kind: "seconds"},
        {path: "welcome.boxAfter", label: "First box after (s)", kind: "seconds"},
    ]},
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

export async function saveRuleSet(boxes: BoxesAdmin, draft: Draft): Promise<void> {
    await boxes.saveRuleSet({set: ruleSetOf(draft)})
}

// A frozen set offers no box, and refuses every claim.
export function offersOf(set: RuleSet): string {
    return set.rules?.frozen ? "Frozen" : "Open"
}

export function boxesOf(rules: Rules | undefined): string {
    if (rules === undefined) return "—"

    return `Every ${paceOf(rules.minInterval, rules.maxInterval)}, ${rules.maxBoxesPerHour ?? 0} an hour at most`
}
