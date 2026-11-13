import {QuizzesAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {RuleSet} from "../gen/grpc/quizzes/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/quizzes/v1/rules_pb.ts"
import * as ruleBook from "./ruleBook.ts"
import {Book, Draft, Group} from "./ruleBook.ts"
import {paceOf, Switch} from "./ruleValues.ts"

// The quizzes' rule sets (quizzes.v1): when a quiz comes, how long it stays and how its question is drawn. What a right
// answer grants is the offers' (bonusRules.ts).

export type {Draft}

// No quiz offered, and every answer refused, while it plays.
export const FROZEN: Switch = {path: "frozen", label: "Frozen: no quiz is offered or answered", short: "Frozen"}

export const GROUPS: Group[] = [
    {
        title: "Quizzes",
        switches: [{path: "enabled", label: "Quizzes on", short: "Quizzes on"}],
        fields: [
            {path: "minInterval", label: "Shortest quiz wait (s)", kind: "seconds"},
            {path: "maxInterval", label: "Longest quiz wait (s)", kind: "seconds"},
            {path: "maxRightAnswersPerHour", label: "Right answers an hour, at most", kind: "whole"},
            {path: "offerTtl", label: "Banner (s)", kind: "seconds"},
            {path: "answerWindow", label: "Time to answer (s)", kind: "seconds"},
            {path: "missRetry", label: "Next quiz after a miss (s)", kind: "seconds"},
            {path: "leaderBias", label: "Lean to the leaders", kind: "number"},
            {path: "activeWithin", label: "Last click within (s)", kind: "seconds"},
        ],
    },
    {title: "Quiz difficulty", switches: [], fields: [
        {path: "width", label: "Bell width, 0 is off", kind: "number"},
        {path: "target", label: "Share right aimed at", kind: "number"},
        {path: "floor", label: "Never asked under, share right", kind: "number"},
        {path: "floorAfter", label: "Floor after, answers", kind: "whole"},
        {path: "explore", label: "Draws that ignore it, share", kind: "number"},
    ]},
    {title: "New accounts", switches: [], fields: [
        {path: "welcome.within", label: "New for (s)", kind: "seconds"},
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

export async function saveRuleSet(quizzes: QuizzesAdmin, draft: Draft): Promise<void> {
    await quizzes.saveRuleSet({set: ruleSetOf(draft)})
}

// A frozen set offers no quiz, and refuses every answer.
export function offersOf(set: RuleSet): string {
    return set.rules?.frozen ? "Frozen" : "Open"
}

export function quizzesOf(rules: Rules | undefined): string {
    if (rules === undefined) return "—"
    if (rules.enabled !== true) return "Off"

    return `Every ${paceOf(rules.minInterval, rules.maxInterval)}, ${rules.maxRightAnswersPerHour ?? 0} right an hour at most`
}
