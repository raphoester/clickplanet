import {SeasonsAdmin} from "../clients.ts"
import {RuleSet} from "../gen/grpc/seasons/v1/admin_pb.ts"
import {OwnRoundRules, Rules} from "../gen/grpc/seasons/v1/rules_pb.ts"

// The server refuses more (seasons' rules.MaxTimes): a slip of the keyboard cannot decide a season.
export const MAX_TIMES = 10

// A set as the form edits it: whether the window of the entry that plays it is a round of its own, and its points'
// times, a field's text until it is saved.
export type Draft = {name: string, ownRound: boolean, times: string}

export function draftOf(set: RuleSet): Draft {
    return {name: set.name, ownRound: set.rules?.ownRound?.enabled === true, times: String(set.rules?.ownRound?.times ?? 1)}
}

// The rules in force change only through the schedule: a new set starts as a copy of another, with no name.
export function copyOf(set: RuleSet): Draft {
    return {...draftOf(set), name: ""}
}

// The times are written when the round is off too: the server reads every value.
export function complete(draft: Draft): boolean {
    const times = Number(draft.times)
    return draft.name.trim() !== "" && Number.isInteger(times) && times >= 1 && times <= MAX_TIMES
}

export async function saveRuleSet(seasons: SeasonsAdmin, draft: Draft): Promise<void> {
    await seasons.saveRuleSet({set: new RuleSet({
        name: draft.name.trim(),
        rules: new Rules({ownRound: new OwnRoundRules({enabled: draft.ownRound, times: Number(draft.times)})}),
    })})
}

// "Of its own, 3× a day's points", or "Days only".
export function roundOf(set: RuleSet): string {
    const own = set.rules?.ownRound
    return own?.enabled === true ? `Of its own, ${own.times}× a day's points` : "Days only"
}
