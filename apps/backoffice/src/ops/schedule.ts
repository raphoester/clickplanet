import {Timestamp} from "@bufbuild/protobuf"
import {SeasonsAdmin} from "../clients.ts"
import {GetRulesResponse as AntibotRules} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {GetRulesResponse as PlanetRules} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Entry, GetScheduleResponse, Phase, Stage} from "../gen/grpc/seasons/v1/admin_pb.ts"

// A line of the schedule: a season's phase, or an operator's entry.
export type Row = {
    key: string
    set: string
    startsAt?: Date
    endsAt?: Date
    priority: number
    phase?: Phase
    entry?: Entry
}

// Phases and entries by start; Season 0's days, which have no start, first.
export function rowsOf(schedule: GetScheduleResponse): Row[] {
    const phases: Row[] = schedule.phases.map(phase => ({
        key: `phase ${phase.season} ${phase.stage}`, set: phase.ruleSet, startsAt: phase.startsAt?.toDate(),
        endsAt: phase.endsAt?.toDate(), priority: 0, phase,
    }))
    const entries: Row[] = schedule.entries.map(entry => ({
        key: entry.id, set: entry.ruleSet, startsAt: entry.startsAt?.toDate(), endsAt: entry.endsAt?.toDate(),
        priority: entry.priority, entry,
    }))
    return [...phases, ...entries].sort((a, b) => (a.startsAt?.getTime() ?? -Infinity) - (b.startsAt?.getTime() ?? -Infinity))
}

// The row the schedule plays now: its entry, or the phase that runs now when no entry plays.
export function playingOf(schedule: GetScheduleResponse, rows: Row[], now: Date): string | undefined {
    const entry = schedule.playing?.entryId ?? ""
    if (entry !== "") return entry
    return rows.find(row => row.phase !== undefined && runs(row, now))?.key
}

function runs(row: Row, now: Date): boolean {
    return (row.startsAt === undefined || row.startsAt <= now) && (row.endsAt === undefined || now < row.endsAt)
}

export function ended(row: Row, now: Date): boolean {
    return row.endsAt !== undefined && row.endsAt <= now
}

// Only an entry to come changes.
export function changeable(row: Row, now: Date): boolean {
    return row.entry !== undefined && row.startsAt !== undefined && now < row.startsAt
}

export function phaseName(phase: Phase): string {
    switch (phase.stage) {
        case Stage.DAYS:
            return `Season ${phase.season}, days`
        case Stage.FINALE:
            return `Season ${phase.season}, finale`
        case Stage.OVER:
            return `Season ${phase.season}, over`
        case Stage.UNSPECIFIED:
            return `Season ${phase.season}`
    }
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
        Number(draft.priority) >= 1
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

export async function addEntry(seasons: SeasonsAdmin, draft: Draft): Promise<void> {
    await seasons.addEntry(valuesOf(draft))
}

export async function editEntry(seasons: SeasonsAdmin, id: string, draft: Draft): Promise<void> {
    await seasons.editEntry({id, ...valuesOf(draft)})
}

export async function deleteEntry(seasons: SeasonsAdmin, id: string): Promise<void> {
    await seasons.deleteEntry({id})
}
