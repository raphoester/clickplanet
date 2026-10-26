import {Timestamp} from "@bufbuild/protobuf"
import {MetagameAdmin} from "../clients.ts"
import {GetRulesResponse as AntibotRules} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {GetRulesResponse as PlanetRules} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Entry, GetScheduleResponse} from "../gen/grpc/metagame/v1/admin_pb.ts"

// A line of the schedule: an entry and its times.
export type Row = {entry: Entry, startsAt?: Date, endsAt?: Date}

// The entries by start.
export function rowsOf(schedule: GetScheduleResponse): Row[] {
    return schedule.entries
        .map(entry => ({entry, startsAt: entry.startsAt?.toDate(), endsAt: entry.endsAt?.toDate()}))
        .sort((a, b) => (a.startsAt?.getTime() ?? 0) - (b.startsAt?.getTime() ?? 0))
}

export function ended(row: Row, now: Date): boolean {
    return row.endsAt !== undefined && row.endsAt <= now
}

// Only an entry to come changes.
export function changeable(row: Row, now: Date): boolean {
    return row.startsAt !== undefined && now < row.startsAt
}

// The modules that keep no set of this name.
export function lacking(set: string, planet: PlanetRules, antibot: AntibotRules): string[] {
    const missing: string[] = []
    if (!planet.sets.some(kept => kept.name === set)) missing.push("planet")
    if (!antibot.sets.some(kept => kept.name === set)) missing.push("the antibot")
    return missing
}

// The sets an entry may play: the ones both modules keep.
export function playable(planet: PlanetRules, antibot: AntibotRules): string[] {
    return planet.sets.map(set => set.name).filter(name => antibot.sets.some(kept => kept.name === name))
}

// An entry as the form edits it: each time a datetime-local, in the operator's own time.
export type Draft = {set: string, startsAt: string, endsAt: string, priority: string}

export function draftOf(entry: Entry | undefined, sets: string[]): Draft {
    if (entry === undefined) return {set: sets.find(set => set !== "default") ?? sets[0] ?? "", startsAt: "", endsAt: "", priority: "1"}
    return {
        set: entry.ruleSet, startsAt: localOf(entry.startsAt?.toDate()), endsAt: localOf(entry.endsAt?.toDate()),
        priority: String(entry.priority),
    }
}

function localOf(date: Date | undefined): string {
    if (date === undefined) return ""
    const pad = (n: number) => String(n).padStart(2, "0")
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// The server holds the rest: a start after now, an end after it, a set both modules keep.
export function complete(draft: Draft): boolean {
    return draft.set !== "" && draft.startsAt !== "" && draft.endsAt !== "" && Number.isInteger(Number(draft.priority)) &&
        Number(draft.priority) >= 0
}

function valuesOf(draft: Draft) {
    return {
        ruleSet: draft.set,
        startsAt: Timestamp.fromDate(new Date(draft.startsAt)),
        endsAt: Timestamp.fromDate(new Date(draft.endsAt)),
        priority: Number(draft.priority),
    }
}

export function questionOf(draft: Draft): string {
    return `Play ${draft.set} from ${new Date(draft.startsAt).toLocaleString()} to ${new Date(draft.endsAt).toLocaleString()}, ` +
        `at priority ${draft.priority}?`
}

export async function addEntry(metagame: MetagameAdmin, draft: Draft): Promise<void> {
    await metagame.addEntry(valuesOf(draft))
}

export async function editEntry(metagame: MetagameAdmin, id: string, draft: Draft): Promise<void> {
    await metagame.editEntry({id, ...valuesOf(draft)})
}

export async function deleteEntry(metagame: MetagameAdmin, id: string): Promise<void> {
    await metagame.deleteEntry({id})
}
