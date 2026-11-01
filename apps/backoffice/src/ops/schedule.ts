import {proto3, Timestamp} from "@bufbuild/protobuf"
import {GamemasterAdmin} from "../clients.ts"
import {localOf} from "../components/time.ts"
import {GetRulesResponse as AntibotRules} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {GetRulesResponse as PlanetRules} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Entry, GetScheduleResponse, ScheduledFinale} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {EntryKind} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {Cutoff, offCutoffWords} from "./cutoff.ts"

// A line of the schedule: an entry and its times.
export type Row = {entry: Entry, startsAt?: Date, endsAt?: Date}

// The entries by start.
export function rowsOf(schedule: GetScheduleResponse): Row[] {
    return schedule.entries
        .map(entry => ({entry, startsAt: entry.startsAt?.toDate(), endsAt: entry.endsAt?.toDate()}))
        .sort((a, b) => (a.startsAt?.getTime() ?? 0) - (b.startsAt?.getTime() ?? 0))
}

// A season's final battle: it plays above every entry, and is changed on its season.
export type FinaleRow = {finale: ScheduledFinale, startsAt?: Date, endsAt?: Date}

// The entries and the seasons' final battles, by start.
export function linesOf(schedule: GetScheduleResponse): (Row | FinaleRow)[] {
    const finales: FinaleRow[] = schedule.finales.map(finale => ({finale, startsAt: finale.startsAt?.toDate(), endsAt: finale.endsAt?.toDate()}))
    return [...rowsOf(schedule), ...finales].sort((a, b) => (a.startsAt?.getTime() ?? 0) - (b.startsAt?.getTime() ?? 0))
}

export function ended(row: {endsAt?: Date}, now: Date): boolean {
    return row.endsAt !== undefined && row.endsAt <= now
}

// Only an entry to come changes.
export function changeable(row: Row, now: Date): boolean {
    return row.startsAt !== undefined && now < row.startsAt
}

// What each module the schedule commands keeps and plays.
export type Modules = {planet: PlanetRules, antibot: AntibotRules}

// The modules that keep no set of this name.
export function lacking(set: string, modules: Modules): string[] {
    const missing: string[] = []
    if (!modules.planet.sets.some(kept => kept.name === set)) missing.push("planet")
    if (!modules.antibot.sets.some(kept => kept.name === set)) missing.push("the antibot")
    return missing
}

// The sets an entry may play: the ones every module keeps.
export function playable(modules: Modules): string[] {
    return modules.planet.sets.map(set => set.name).filter(name => lacking(name, modules).length === 0)
}

// The kinds an entry may be, read off the proto, so a new one shows with no edit here.
export function kinds(): EntryKind[] {
    return proto3.getEnumType(EntryKind).values.map(value => value.no as EntryKind).filter(kind => kind !== EntryKind.UNSPECIFIED)
}

// The kind's word: "battle" for ENTRY_KIND_BATTLE.
export function wordOf(kind: EntryKind): string {
    const value = proto3.getEnumType(EntryKind).findNumber(kind)
    return (value?.localName ?? "unknown").toLowerCase()
}

// The server refuses a longer description (gamemaster.v1.AddEntryRequest.description).
export const MAX_DESCRIPTION = 300

// What the list shows of a description.
export function shortOf(description: string, max = 60): string {
    return description.length <= max ? description : `${description.slice(0, max - 1)}…`
}

// The server's bound (gamemaster.v1.AddEntryRequest.times).
export const MAX_TIMES = 10

// A battle's points' times; a plain entry has none.
export function timesOf(kind: EntryKind, times: string): number {
    return kind === EntryKind.BATTLE ? Number(times) : 0
}

// An entry as the form edits it: each time a datetime-local, in the operator's own time.
export type Draft = {
    set: string, kind: EntryKind, description: string, startsAt: string, endsAt: string, priority: string, times: string,
}

export function draftOf(entry: Entry | undefined, sets: string[]): Draft {
    if (entry === undefined) {
        return {
            set: sets.find(set => set !== "default") ?? sets[0] ?? "", kind: EntryKind.BATTLE, description: "", startsAt: "", endsAt: "",
            priority: "1", times: "3",
        }
    }
    return {
        set: entry.ruleSet, kind: entry.kind, description: entry.description, startsAt: localOf(entry.startsAt?.toDate()),
        endsAt: localOf(entry.endsAt?.toDate()), priority: String(entry.priority),
        times: entry.kind === EntryKind.BATTLE ? String(entry.times) : "3",
    }
}

// The server holds the rest: a start after now, an end after it, a set every module keeps.
export function complete(draft: Draft): boolean {
    return timesFit(draft) && draft.set !== "" && draft.kind !== EntryKind.UNSPECIFIED && [...draft.description].length <= MAX_DESCRIPTION &&
        draft.startsAt !== "" && draft.endsAt !== "" && Number.isInteger(Number(draft.priority)) && Number(draft.priority) >= 0
}

// Why the server will refuse a battle that neither starts nor ends at the cutoff, or undefined.
export function offCutoffOf(draft: Draft, cutoff: Cutoff | undefined): string | undefined {
    const startsAt = new Date(draft.startsAt)
    const endsAt = new Date(draft.endsAt)
    if (draft.kind !== EntryKind.BATTLE || cutoff === undefined || Number.isNaN(startsAt.getTime()) || Number.isNaN(endsAt.getTime())) {
        return undefined
    }
    return offCutoffWords("The battle", startsAt, endsAt, cutoff)
}

function timesFit(draft: Draft): boolean {
    if (draft.kind !== EntryKind.BATTLE) return true
    const times = Number(draft.times)
    return Number.isInteger(times) && times >= 1 && times <= MAX_TIMES
}

function valuesOf(draft: Draft) {
    return {
        ruleSet: draft.set,
        kind: draft.kind,
        description: draft.description,
        startsAt: Timestamp.fromDate(new Date(draft.startsAt)),
        endsAt: Timestamp.fromDate(new Date(draft.endsAt)),
        priority: Number(draft.priority),
        times: timesOf(draft.kind, draft.times),
    }
}

export function questionOf(draft: Draft): string {
    const times = draft.kind === EntryKind.BATTLE ? ` worth ${draft.times}× points` : ""
    return `Play ${draft.set} as a ${wordOf(draft.kind)} entry${times} from ${new Date(draft.startsAt).toLocaleString()} to ` +
        `${new Date(draft.endsAt).toLocaleString()}, at priority ${draft.priority}?`
}

export async function addEntry(gamemaster: GamemasterAdmin, draft: Draft): Promise<void> {
    await gamemaster.addEntry(valuesOf(draft))
}

export async function editEntry(gamemaster: GamemasterAdmin, id: string, draft: Draft): Promise<void> {
    await gamemaster.editEntry({id, ...valuesOf(draft)})
}

export async function deleteEntry(gamemaster: GamemasterAdmin, id: string): Promise<void> {
    await gamemaster.deleteEntry({id})
}
