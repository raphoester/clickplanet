import {Entry, GetScheduleResponse, GetSeasonsResponse} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {EntryKind} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {GetRulesResponse as PlanetRules} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordOf} from "./schedule.ts"

export type Lane = "seasons" | "rules" | "battles" | "days"

export const LANES: {lane: Lane, label: string}[] = [
    {lane: "seasons", label: "Seasons"},
    {lane: "rules", label: "Rules in force"},
    {lane: "battles", label: "Battles"},
    {lane: "days", label: "Days"},
]

export type Tone = "season" | "break" | "finale" | "battle" | "plain" | "frozen" | "default" | "day"

export type Block = {lane: Lane, label: string, detail: string, tone: Tone, startsAt: Date, endsAt: Date}

export type Happening = {what: string, tone: Tone}

// Everything that happens at one time, in the order the gamemaster does it.
export type Instant = {at: Date, happenings: Happening[]}

export type Timeline = {blocks: Block[], instants: Instant[]}

type Inputs = {schedule: GetScheduleResponse, seasons: GetSeasonsResponse, planet: PlanetRules}

const DEFAULT = "default"

export function timelineOf({schedule, seasons, planet}: Inputs): Timeline {
    const frozen = new Set(planet.sets.filter(set => set.rules?.frozen === true).map(set => set.name))
    const blocks = [...seasonBlocks(seasons), ...ruleBlocks(schedule, frozen), ...battleBlocks(schedule), ...dayBlocks(seasons)]
    return {blocks, instants: instantsOf(blocks, seasons)}
}

function seasonBlocks(seasons: GetSeasonsResponse): Block[] {
    const blocks: Block[] = []
    let previousEnd: Date | undefined
    for (const scheduled of seasons.seasons) {
        const season = scheduled.season
        const startsAt = season?.startsAt?.toDate()
        const endsAt = season?.endsAt?.toDate()
        if (season === undefined || startsAt === undefined || endsAt === undefined) continue
        if (previousEnd !== undefined && previousEnd < startsAt) {
            blocks.push({lane: "seasons", label: "Break", detail: `Before season ${season.number}`, tone: "break", startsAt: previousEnd, endsAt: startsAt})
        }
        blocks.push({lane: "seasons", label: `Season ${season.number}`, detail: "", tone: "season", startsAt, endsAt})
        previousEnd = endsAt
    }
    return blocks
}

function dayBlocks(seasons: GetSeasonsResponse): Block[] {
    return seasons.seasons.flatMap(scheduled => scheduled.days.flatMap(day => {
        const startsAt = day.startsAt?.toDate()
        const endsAt = day.endsAt?.toDate()
        if (startsAt === undefined || endsAt === undefined) return []
        const label = `Day ${day.number}`
        return [{lane: "days" as const, label, detail: `Season ${scheduled.season?.number ?? 0}`, tone: "day" as const, startsAt, endsAt}]
    }))
}

function battleBlocks(schedule: GetScheduleResponse): Block[] {
    const battles: Block[] = schedule.entries.filter(entry => entry.kind === EntryKind.BATTLE).flatMap(entry => {
        const startsAt = entry.startsAt?.toDate()
        const endsAt = entry.endsAt?.toDate()
        if (startsAt === undefined || endsAt === undefined) return []
        return [{lane: "battles" as const, label: `Battle ×${entry.times}`, detail: entry.ruleSet, tone: "battle" as const, startsAt, endsAt}]
    })
    const finales: Block[] = schedule.finales.flatMap(scheduled => {
        const startsAt = scheduled.startsAt?.toDate()
        const endsAt = scheduled.endsAt?.toDate()
        if (startsAt === undefined || endsAt === undefined) return []
        const label = `Final Battle ×${scheduled.finale?.times ?? 0}`
        return [{lane: "battles" as const, label, detail: `Season ${scheduled.season}, ${scheduled.finale?.ruleSet ?? ""}`, tone: "finale" as const, startsAt, endsAt}]
    })
    return [...battles, ...finales]
}

// What plays over time, by the gamemaster's rule: a season's finale above all, else the running entry of the highest
// priority (at one priority, the one made last), else default.
type Playing = {set: string, source: "finale" | "entry" | "default", kind?: EntryKind, season?: number}

function ruleBlocks(schedule: GetScheduleResponse, frozen: Set<string>): Block[] {
    const edges = new Set<number>()
    for (const span of [...schedule.entries, ...schedule.finales]) {
        const startsAt = span.startsAt?.toDate().getTime()
        const endsAt = span.endsAt?.toDate().getTime()
        if (startsAt !== undefined) edges.add(startsAt)
        if (endsAt !== undefined) edges.add(endsAt)
    }
    const times = [...edges].sort((a, b) => a - b)
    const blocks: Block[] = []
    for (let i = 0; i + 1 < times.length; i++) {
        const playing = playingAt(schedule, new Date((times[i] + times[i + 1]) / 2))
        const last = blocks[blocks.length - 1]
        if (last !== undefined && last.endsAt.getTime() === times[i] && last.label === playing.set && last.detail === detailOf(playing)) {
            last.endsAt = new Date(times[i + 1])
            continue
        }
        blocks.push({
            lane: "rules", label: playing.set, detail: detailOf(playing), tone: toneOf(playing, frozen),
            startsAt: new Date(times[i]), endsAt: new Date(times[i + 1]),
        })
    }
    return blocks.filter(block => block.tone !== "default")
}

function playingAt(schedule: GetScheduleResponse, at: Date): Playing {
    const finale = schedule.finales.find(scheduled => runs(scheduled, at))
    if (finale !== undefined) return {set: finale.finale?.ruleSet ?? DEFAULT, source: "finale", season: finale.season}
    const entry = schedule.entries.filter(entry => runs(entry, at)).sort(above)[0]
    if (entry !== undefined) return {set: entry.ruleSet, source: "entry", kind: entry.kind}
    return {set: DEFAULT, source: "default"}
}

function above(a: Entry, b: Entry): number {
    if (a.priority !== b.priority) return b.priority - a.priority
    return (b.madeAt?.toDate().getTime() ?? 0) - (a.madeAt?.toDate().getTime() ?? 0)
}

function runs(span: {startsAt?: {toDate(): Date}, endsAt?: {toDate(): Date}}, at: Date): boolean {
    const startsAt = span.startsAt?.toDate()
    const endsAt = span.endsAt?.toDate()
    return startsAt !== undefined && endsAt !== undefined && startsAt <= at && at < endsAt
}

function detailOf(playing: Playing): string {
    switch (playing.source) {
        case "finale":
            return `Season ${playing.season}'s final battle`
        case "entry":
            return `${capitalized(wordOf(playing.kind ?? EntryKind.PLAIN))} entry`
        case "default":
            return ""
    }
}

function capitalized(word: string): string {
    return word.charAt(0).toUpperCase() + word.slice(1)
}

function toneOf(playing: Playing, frozen: Set<string>): Tone {
    if (frozen.has(playing.set)) return "frozen"
    switch (playing.source) {
        case "finale":
            return "finale"
        case "entry":
            return playing.kind === EntryKind.BATTLE ? "battle" : "plain"
        case "default":
            return "default"
    }
}

// The order the gamemaster does things at one time: a round closes before its book, a book opens before its rounds.
type Step = "dayCloses" | "battleEnds" | "seasonEnds" | "seasonStarts" | "mapResets" | "battleStarts" | "rulesPlay"
const ORDER: Step[] = ["dayCloses", "battleEnds", "seasonEnds", "seasonStarts", "mapResets", "battleStarts", "rulesPlay"]

function instantsOf(blocks: Block[], seasons: GetSeasonsResponse): Instant[] {
    const at = new Map<number, (Happening & {step: Step})[]>()
    const add = (date: Date, step: Step, happening: Happening) => {
        const list = at.get(date.getTime()) ?? []
        list.push({...happening, step})
        at.set(date.getTime(), list)
    }
    const firstSeason = seasons.seasons[0]?.season?.number

    for (const block of blocks) {
        switch (block.lane) {
            case "seasons":
                if (block.tone !== "season") break
                add(block.startsAt, "seasonStarts", {what: `${block.label} starts`, tone: "season"})
                if (block.label !== `Season ${firstSeason}`) add(block.startsAt, "mapResets", {what: "The map is reset", tone: "season"})
                add(block.endsAt, "seasonEnds", {what: `${block.label} ends`, tone: "season"})
                break
            case "days":
                add(block.endsAt, "dayCloses", {what: `${block.label} closes`, tone: "day"})
                break
            case "battles":
                add(block.startsAt, "battleStarts", {what: `${block.label} starts (${block.detail})`, tone: block.tone})
                add(block.endsAt, "battleEnds", {what: `${block.label} ends`, tone: block.tone})
                break
            case "rules":
                add(block.startsAt, "rulesPlay", {what: `${block.label} plays${block.tone === "frozen" ? ": the map is frozen" : ""}`, tone: block.tone})
                break
        }
    }
    const rules = blocks.filter(block => block.lane === "rules")
    for (const block of rules) {
        const next = rules.find(other => other.startsAt.getTime() === block.endsAt.getTime())
        if (next === undefined) add(block.endsAt, "rulesPlay", {what: `${DEFAULT} plays`, tone: "default"})
    }

    return [...at.entries()]
        .sort(([a], [b]) => a - b)
        .map(([time, happenings]) => ({
            at: new Date(time),
            happenings: happenings
                .sort((a, b) => ORDER.indexOf(a.step) - ORDER.indexOf(b.step))
                .map(({what, tone}) => ({what, tone})),
        }))
}

export function upcoming(instants: Instant[], from: Date): Instant[] {
    return instants.filter(instant => instant.at >= from)
}

// A block's place along a range, in percent, clipped to it; undefined when it is out of the range.
export function placeOf(block: {startsAt: Date, endsAt: Date}, from: Date, to: Date): {left: number, width: number} | undefined {
    const span = to.getTime() - from.getTime()
    const start = Math.max(block.startsAt.getTime(), from.getTime())
    const end = Math.min(block.endsAt.getTime(), to.getTime())
    if (end <= start) return undefined
    return {left: (start - from.getTime()) / span * 100, width: (end - start) / span * 100}
}

// "in 3 h 20 min", "in 2 d 4 h", "now".
export function untilOf(at: Date, now: Date): string {
    const minutes = Math.round((at.getTime() - now.getTime()) / 60000)
    if (minutes <= 0) return "now"
    const days = Math.floor(minutes / 1440)
    const hours = Math.floor((minutes % 1440) / 60)
    const rest = minutes % 60
    if (days > 0) return `in ${days} d${hours > 0 ? ` ${hours} h` : ""}`
    if (hours > 0) return `in ${hours} h${rest > 0 ? ` ${rest} min` : ""}`
    return `in ${rest} min`
}
