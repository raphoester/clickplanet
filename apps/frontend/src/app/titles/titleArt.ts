import {PlayerTitle, TitleTrack} from "../../backends/player.ts"

export type Metal = "bronze" | "silver" | "gold" | "platinum" | "prism" | "holo"

export const METALS: Record<Metal, readonly string[]> = {
    bronze: ["#F6C59A", "#C97B45", "#6E3615"],
    silver: ["#FFFFFF", "#C3CBD6", "#6F7A88"],
    gold: ["#FFF3BF", "#F2B632", "#8F5605"],
    platinum: ["#F4F9FF", "#A9C1DD", "#4E6178"],
    prism: ["#8EF6FF", "#9C7BFF", "#FF7AD0", "#FFD36E"],
    holo: ["#B6FFE9", "#8FB8FF", "#C79BFF", "#FFE59A"],
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

export function enamelOf(title: PlayerTitle): string {
    if (title.rank?.trackId === "devotion") return "#3A1606"
    if (title.rank?.trackId === "chatter") return "#0B2E2B"
    if (title.rank) return "#14223D"
    return "#1F1238"
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
