import {MissionsAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {FlagMissionRules, Rules} from "../gen/grpc/missions/v1/rules_pb.ts"

export type FieldKey = "goalPercent" | "leastGoal" | "mostPoints" | "fewestPoints"

// Every number of a flag mission, with its label: a test fails when missions.v1.FlagMissionRules gains one the form
// does not name.
export const FIELDS: {key: FieldKey, label: string}[] = [
    {key: "goalPercent", label: "Goal, % of all the flag did the day before"},
    {key: "leastGoal", label: "Least goal, tiles or shields"},
    {key: "mostPoints", label: "Points for a flag holding no tile"},
    {key: "fewestPoints", label: "Points for the biggest flag"},
]

export type Draft = Record<FieldKey, string>

export function draftOf(rules: Rules | undefined): Draft {
    const flags = rules?.flagMissions ?? new FlagMissionRules()
    return {
        goalPercent: String(flags.goalPercent), leastGoal: String(flags.leastGoal), mostPoints: String(flags.mostPoints),
        fewestPoints: String(flags.fewestPoints),
    }
}

// The first number left empty or not whole, by its label. The server checks the rest, and says each.
export function flawOf(draft: Draft): string | undefined {
    const field = FIELDS.find(each => !/^\d+$/.test(draft[each.key].trim()))
    return field === undefined ? undefined : `${field.label}: a whole number.`
}

export function changesOf(kept: Draft, draft: Draft): Change[] {
    return FIELDS
        .filter(field => kept[field.key] !== draft[field.key].trim())
        .map(field => ({label: field.label, before: kept[field.key], after: draft[field.key].trim()}))
}

export function rulesOf(draft: Draft): Rules {
    return new Rules({flagMissions: new FlagMissionRules({
        goalPercent: Number(draft.goalPercent), leastGoal: Number(draft.leastGoal), mostPoints: Number(draft.mostPoints),
        fewestPoints: Number(draft.fewestPoints),
    })})
}

export async function saveRules(missions: MissionsAdmin, draft: Draft): Promise<void> {
    await missions.saveRules({rules: rulesOf(draft)})
}
