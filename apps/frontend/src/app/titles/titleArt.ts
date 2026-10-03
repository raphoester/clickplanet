import {PlayerTitle, TitleTrack} from "../../backends/player.ts"

export type Metal = "bronze" | "silver" | "gold" | "platinum" | "prism" | "holo"

type Track = "conquest" | "chatter" | "devotion" | "og"

export const METALS: Record<Metal, string> = {
    bronze: "var(--bronze)",
    silver: "var(--silver)",
    gold: "var(--gold)",
    platinum: "var(--platinum)",
    prism: "color-mix(in srgb, var(--prism-2) 70%, var(--text))",
    holo: "color-mix(in srgb, var(--holo-1) 50%, var(--holo-2))",
}

export const BANDS: Partial<Record<Metal, readonly string[]>> = {
    prism: ["var(--prism-1)", "var(--prism-2)", "var(--prism-3)", "var(--prism-4)"],
    holo: ["var(--holo-1)", "var(--holo-2)", "var(--holo-3)", "var(--holo-4)"],
}

const TITLE_METALS: Partial<Record<string, Metal>> = {
    og: "holo",
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
}

export function metalOf(title: PlayerTitle): Metal {
    return TITLE_METALS[title.id] ?? "silver"
}

function trackOf(title: PlayerTitle): Track {
    if (title.rank?.trackId === "devotion") return "devotion"
    if (title.rank?.trackId === "chatter") return "chatter"
    if (title.rank) return "conquest"
    return "og"
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
}

const count = new Intl.NumberFormat()

export function stepLabel(trackId: string, threshold: number): string {
    return TRACK_UNITS[trackId]?.step(count.format(threshold)) ?? count.format(threshold)
}

export function leftLabel(trackId: string, remaining: number, next: string, progress: number): string {
    return TRACK_UNITS[trackId]?.left(count.format(remaining), next, count.format(progress))
        ?? `${count.format(remaining)} to ${next}`
}
