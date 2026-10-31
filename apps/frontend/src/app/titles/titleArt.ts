import {PlayerTitle, TitleTrack} from "../../backends/player.ts"

export type Metal = "bronze" | "silver" | "gold" | "platinum" | "prism" | "holo" | "pumpkin"

const TRACKS = ["conquest", "chatter", "devotion", "defense", "bombs", "loot", "quiz", "missions"] as const

type Track = typeof TRACKS[number] | "og" | "halloween"

const STANDALONE_LOOKS: Partial<Record<string, Track>> = {
    halloween_2026: "halloween",
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

function trackOf(title: PlayerTitle): Track {
    if (!title.rank) return STANDALONE_LOOKS[title.id] ?? "og"
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
    return title.rank && `Rank ${title.rank.number} of ${title.rank.count} · ${title.rank.trackName}`
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

const TRACK_UNITS: Partial<Record<string, {step: (n: string) => string, left: (n: string, next: string, progress: string) => string}>> = {
    conquest: {
        step: (n) => `${n} tiles`,
        left: (n, next) => `${n} tiles to ${next}`,
    },
    devotion: {
        step: (n) => `${n} days in a row`,
        left: (n, next, progress) => `Day ${progress} · ${n} more to ${next}`,
    },
    chatter: {
        step: (n) => `${n} messages`,
        left: (n, next) => `${n} messages to ${next}`,
    },
    defense: {
        step: (n) => `${n} fortified tiles`,
        left: (n, next) => `${n} fortified tiles to ${next}`,
    },
    bombs: {
        step: (n) => `${n} ${n === "1" ? "bomb" : "bombs"}`,
        left: (n, next) => `${n} ${n === "1" ? "bomb" : "bombs"} to ${next}`,
    },
    loot: {
        step: (n) => `${n} boxes`,
        left: (n, next) => `${n} ${n === "1" ? "box" : "boxes"} to ${next}`,
    },
    quiz: {
        step: (n) => `${n} right answers`,
        left: (n, next) => `${n} ${n === "1" ? "right answer" : "right answers"} to ${next}`,
    },
    missions: {
        step: (n) => `${n} ${n === "1" ? "mission" : "missions"}`,
        left: (n, next) => `${n} ${n === "1" ? "mission" : "missions"} to ${next}`,
    },
}

const count = new Intl.NumberFormat()

export function stepLabel(trackId: string, threshold: number): string {
    return TRACK_UNITS[trackId]?.step(count.format(threshold)) ?? count.format(threshold)
}

export function leftLabel(trackId: string, remaining: number, next: string, progress: number): string {
    return TRACK_UNITS[trackId]?.left(count.format(remaining), next, count.format(progress))
        ?? `${count.format(remaining)} to ${next}`
}
