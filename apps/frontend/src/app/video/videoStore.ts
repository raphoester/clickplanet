import {create} from "zustand"
import {dotsAfter, dotsTapped, DotTrigger, parseDots, VideoDots} from "../../domain/video/dots.ts"
import {Ground} from "../../domain/video/story.ts"
import {MapHistory} from "../../domain/mapHistory.ts"
import {RecentHistorySource} from "../../backends/recentHistory.ts"
import {countryOfTile, loadBorders} from "../viewer/borderField.ts"
import {BORDERS_URL} from "../viewer/bordersAsset.ts"

const DOTS_KEY = "clickplanet-video-dots"
const CLICKED_KEY = "clickplanet-video-clicked"

// The history's frames are 30 minutes apart, on :00 and :30: a newer one is out once a step has passed.
const STEP_MS = 30 * 60_000

export type Read = {history: MapHistory, ground: Ground}

type VideoStore = {
    read?: Read
    dots: VideoDots
    // The player's day of the last click this browser made.
    clicked?: string
    open: boolean
    learn: (read: Read) => void
    clickedOn: (day: string) => void
    light: (trigger: DotTrigger, today: string, moved: number) => void
    tap: () => void
    show: () => void
    close: () => void
}

function stored(key: string): string | null {
    try {
        return window.localStorage.getItem(key)
    } catch {
        return null
    }
}

function store(key: string, value: string) {
    try {
        window.localStorage.setItem(key, value)
    } catch {
        // storage unavailable: the dots and the click last as long as the page
    }
}

function keptDots(state: {dots: VideoDots}, dots: VideoDots) {
    if (dots !== state.dots) store(DOTS_KEY, JSON.stringify(dots))
    return dots
}

export const useVideo = create<VideoStore>()((set) => ({
    dots: parseDots(stored(DOTS_KEY)),
    clicked: stored(CLICKED_KEY) ?? undefined,
    open: false,
    learn: (read) => set({read}),
    clickedOn: (day) => set((state) => {
        if (state.clicked === day) return state
        store(CLICKED_KEY, day)
        return {clicked: day}
    }),
    light: (trigger, today, moved) => set((state) => ({dots: keptDots(state, dotsAfter(state.dots, trigger, today, moved))})),
    tap: () => set((state) => ({dots: keptDots(state, dotsTapped(state.dots)), open: true})),
    show: () => set({open: true}),
    close: () => set({open: false}),
}))

// Read the 24 hours again only once a newer step is out; the browser's cache serves a repeat within the step.
export function stale(read: Read | undefined, now: number): boolean {
    const last = read?.history.frames.at(-1)?.at
    return last === undefined || Math.floor(now / STEP_MS) * STEP_MS > last
}

let borders: Promise<Ground> | undefined

function groundRead(): Promise<Ground> {
    borders ??= loadBorders(BORDERS_URL).then((data) => ({tiles: data.assignment.length, groundOf: (tile) => countryOfTile(data, tile)}))
    return borders
}

let reading: Promise<Read> | undefined

export function readHistory(source: RecentHistorySource): Promise<Read> {
    reading ??= Promise.all([source.recentHistory(), groundRead()])
        .then(([history, ground]) => {
            const read = {history, ground}
            useVideo.getState().learn(read)
            return read
        })
        .finally(() => {
            reading = undefined
        })
    return reading
}
