import {PlayerTitle, TitleTrack} from "../../backends/progression.ts"
import {Key} from "../../i18n/translator.ts"
import {t, translator} from "../../i18n/page.ts"

export type Metal = "bronze" | "silver" | "gold" | "platinum" | "prism" | "holo" | "pumpkin"

const TRACKS = ["conquest", "chatter", "devotion", "defense", "bombs", "loot", "quiz", "missions"] as const

type Track = typeof TRACKS[number] | "og" | "halloween" | "season"

const STANDALONE_LOOKS: Partial<Record<string, Track>> = {
    halloween_2026: "halloween",
}

const SEASONS: Partial<Record<string, string>> = {
    season_0_gold: "0",
    season_0_silver: "0",
    season_0_bronze: "0",
    season_0_top_1: "0",
    season_0_top_2: "0",
    season_0_top_3: "0",
    season_0_battle_top_1: "0",
    season_0_battle_top_2: "0",
    season_0_battle_top_3: "0",
    season_0_veteran: "0",
}

export const METALS: Record<Metal, string> = {
    bronze: "var(--bronze)",
    silver: "var(--silver)",
    gold: "var(--gold)",
    platinum: "var(--platinum)",
    prism: "color-mix(in srgb, var(--prism-2) 70%, var(--text))",
    holo: "color-mix(in srgb, var(--holo-1) 50%, var(--holo-2))",
    pumpkin: "var(--pumpkin)",
}

export const BANDS: Partial<Record<Metal, readonly string[]>> = {
    prism: ["var(--prism-1)", "var(--prism-2)", "var(--prism-3)", "var(--prism-4)"],
    holo: ["var(--holo-1)", "var(--holo-2)", "var(--holo-3)", "var(--holo-4)"],
    pumpkin: Array.from({length: 8}, (_, i) => i % 2 === 0 ? "var(--pumpkin)" : "var(--pumpkin-shade)"),
}

const TITLE_METALS: Partial<Record<string, Metal>> = {
    og: "holo",
    season_0_gold: "gold",
    season_0_silver: "silver",
    season_0_bronze: "bronze",
    season_0_top_1: "gold",
    season_0_top_2: "silver",
    season_0_top_3: "bronze",
    season_0_battle_top_1: "gold",
    season_0_battle_top_2: "silver",
    season_0_battle_top_3: "bronze",
    season_0_veteran: "platinum",
    halloween_2026: "pumpkin",
    settler: "bronze",
    raider: "silver",
    warlord: "gold",
    conqueror: "platinum",
    warmaster: "prism",
    loyal: "bronze",
    devoted: "gold",
    unbroken: "prism",
    talker: "bronze",
    chatterbox: "silver",
    socialite: "gold",
    icon: "prism",
    guard: "bronze",
    defender: "silver",
    protector: "gold",
    warden: "prism",
    bomber: "bronze",
    blaster: "silver",
    wrecker: "gold",
    destroyer: "prism",
    looter: "bronze",
    hunter: "silver",
    pirate: "gold",
    captain: "prism",
    student: "bronze",
    expert: "silver",
    professor: "gold",
    genius: "prism",
    rookie: "bronze",
    agent: "silver",
    operative: "gold",
    spymaster: "prism",
}

export function metalOf(title: PlayerTitle): Metal {
    return TITLE_METALS[title.id] ?? "silver"
}

export function seasonOf(title: PlayerTitle): string | undefined {
    return SEASONS[title.id]
}

function trackOf(title: PlayerTitle): Track {
    if (!title.rank) return seasonOf(title) ? "season" : STANDALONE_LOOKS[title.id] ?? "og"
    return TRACKS.find((track) => track === title.rank?.trackId) ?? "conquest"
}

export function enamelOf(title: PlayerTitle): string {
    return `var(--${trackOf(title)}-enamel)`
}

export function ribbonOf(title: PlayerTitle): string {
    return `var(--${trackOf(title)})`
}

export const OG = "og"

export function rankLine(title: PlayerTitle): string | undefined {
    return title.rank && t("title.rank", {rank: title.rank.number, count: title.rank.count, track: title.rank.trackName})
}

export function filledOf(track: TitleTrack): number {
    const steps = track.steps
    if (steps.length < 2) return steps.length === 1 && steps[0].earned ? 1 : 0

    const next = steps.findIndex((step) => !step.earned)
    const reached = (next === -1 ? steps.length : next) - 1
    if (reached < 0) return 0
    if (reached >= steps.length - 1) return 1

    const from = steps[reached].threshold
    const to = steps[reached + 1].threshold
    const within = Math.min(1, Math.max(0, (track.progress - from) / (to - from)))
    return (reached + within) / (steps.length - 1)
}

const TRACK_LINES: Partial<Record<string, {step: Key, left: Key}>> = {
    conquest: {step: "title.tiles", left: "title.tilesTo"},
    devotion: {step: "title.daysInARow", left: "title.dayMoreTo"},
    chatter: {step: "title.messages", left: "title.messagesTo"},
    defense: {step: "title.fortifiedTiles", left: "title.fortifiedTilesTo"},
    bombs: {step: "title.bombs", left: "title.bombsTo"},
    loot: {step: "title.boxes", left: "title.boxesTo"},
    quiz: {step: "title.rightAnswers", left: "title.rightAnswersTo"},
    missions: {step: "title.missions", left: "title.missionsTo"},
}

export function stepLabel(trackId: string, threshold: number): string {
    const lines = TRACK_LINES[trackId]
    return lines ? t(lines.step, {count: threshold}) : translator().number(threshold)
}

export function leftLabel(trackId: string, remaining: number, next: string, progress: number): string {
    return t(TRACK_LINES[trackId]?.left ?? "title.countTo", {count: remaining, next, day: progress})
}
