import {DOT_SHARE} from "./story.ts"

// The red dots on the video's tile and on the Share tab, as this browser keeps them. lit: the day they last lit.
// visited, moment: the day of the first visit and of the first moment that were read.
export type VideoDots = {
    on: boolean
    lit?: string
    visited?: string
    moment?: string
}

export type DotTrigger = "visit" | "moment"

export const NO_DOTS: VideoDots = {on: false}

// At most once a day, at the first visit or the first moment of the day, and only when the flag gained or lost a tenth
// of a country. moved: that share.
export function dotsAfter(dots: VideoDots, trigger: DotTrigger, today: string, moved: number): VideoDots {
    const seen = trigger === "visit" ? dots.visited : dots.moment
    if (seen === today) return dots
    const marked = trigger === "visit" ? {...dots, visited: today} : {...dots, moment: today}
    if (moved < DOT_SHARE || dots.lit === today) return marked
    return {...marked, on: true, lit: today}
}

// Only a tap on the tile puts them out.
export function dotsTapped(dots: VideoDots): VideoDots {
    return dots.on ? {...dots, on: false} : dots
}

export function parseDots(kept: string | null): VideoDots {
    if (!kept) return NO_DOTS
    try {
        const read = JSON.parse(kept) as Partial<VideoDots>
        return {
            on: read.on === true,
            ...typeof read.lit === "string" ? {lit: read.lit} : {},
            ...typeof read.visited === "string" ? {visited: read.visited} : {},
            ...typeof read.moment === "string" ? {moment: read.moment} : {},
        }
    } catch {
        return NO_DOTS
    }
}

// The player's own day, as its clock reads it.
export function dayOf(at: number): string {
    const date = new Date(at)
    const pad = (value: number) => String(value).padStart(2, "0")
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}
