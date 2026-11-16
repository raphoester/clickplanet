import {BonusesAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import * as ruleBook from "./ruleBook.ts"
import {Book, Draft, Group} from "./ruleBook.ts"

// The bonuses module's rule sets (bonuses.v1): what a box caught or a right answer may be and how much it grants, and
// the sign-in refill. What a hand holds and what a charge does are planet's (the Game rules' Charges); when a box comes
// is the boxes' (boxRules.ts), when a quiz comes the quizzes'.

export type {Draft}

export const GROUPS: Group[] = [
    {title: "What a box or a right answer is", switches: [], fields: [
        {path: "kinds.refill", label: "Refill, weight", kind: "number"},
        {path: "kinds.spreadClicks", label: "Spread, weight", kind: "number"},
        {path: "kinds.encloseClicks", label: "Enclose, weight", kind: "number"},
        {path: "kinds.bomb", label: "Bomb, weight", kind: "number"},
        {path: "kinds.shields", label: "Shields, weight", kind: "number"},
        {path: "kinds.virus", label: "Virus, weight", kind: "number"},
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
        fields: [],
    },
]

export const BOOK: Book<RuleSet> = {
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

const KINDS: {path: keyof NonNullable<Rules["kinds"]>, short: string}[] = [
    {path: "refill", short: "refill"}, {path: "spreadClicks", short: "spread"}, {path: "encloseClicks", short: "enclose"},
    {path: "bomb", short: "bomb"}, {path: "shields", short: "shields"}, {path: "virus", short: "virus"},
]

// The kinds a box or a right answer may be, and each one's weight: "refill 5, spread 3, shields 4".
export function kindsOf(rules: Rules | undefined): string {
    const weighed = KINDS.flatMap(({path, short}) => {
        const weight = rules?.kinds?.[path]
        return weight === undefined ? [] : [`${short} ${weight}`]
    })
    return weighed.length === 0 ? "—" : weighed.join(", ")
}
