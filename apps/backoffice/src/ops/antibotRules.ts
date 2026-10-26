import {AntibotAdmin} from "../clients.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/antibot/v1/rules_pb.ts"

export const DEFAULT_SET = "default"

// A set as the form edits it: its rules as protojson, every value written.
export type Draft = {name: string, text: string}

export function draftOf(set: RuleSet): Draft {
    return {name: set.name, text: textOf(set.rules)}
}

// A new set starts as a copy of default.
export function newDraft(rules: GetRulesResponse): Draft {
    return {name: "", text: textOf(rules.sets.find(set => set.name === DEFAULT_SET)?.rules)}
}

function textOf(rules: Rules | undefined): string {
    return (rules ?? new Rules()).toJsonString({emitDefaultValues: true, prettySpaces: 2})
}

// Why the text is no antibot.v1.Rules, or undefined when it is one. The server holds the bounds a set must keep.
export function flawOf(text: string): string | undefined {
    try {
        Rules.fromJsonString(text)
        return undefined
    } catch (err) {
        return err instanceof Error ? err.message : String(err)
    }
}

export async function saveRuleSet(antibot: AntibotAdmin, draft: Draft): Promise<void> {
    await antibot.saveRuleSet({set: new RuleSet({name: draft.name.trim(), rules: Rules.fromJsonString(draft.text)})})
}

export function bansOf(set: RuleSet): string {
    return set.rules?.bans?.enforce === true ? "Enforced" : "Not enforced"
}

export function crowdOf(set: RuleSet): string {
    return `${set.rules?.cohort?.certainMembers ?? 0} scopes`
}
