import {MapFrame, MapHistory} from "../mapHistory.ts"

// The map's last 24 hours as planet's GetRecentMapHistory tells them: each change with the account that made it, ""
// for none (the first frame, a deleted account).
export type RecentFrame = MapFrame & {accounts: string[]}

export type RecentHistory = Omit<MapHistory, "frames"> & {frames: RecentFrame[]}

export type StoryKind = "gain" | "loss" | "quiet"

// The flag's biggest change of the 24 hours, in the one real country where it gained or lost the most tiles: the
// country's tiles, and the flag's there at the start and at the end.
export type VideoStory = {
    flag: string
    country: string
    kind: StoryKind
    tiles: number
    start: number
    end: number
}

export type Ground = {
    tiles: number
    groundOf: (tile: number) => string | undefined
}

// A change under a hundredth of its country is a quiet day: "TOOK 0%" says nothing.
const QUIET_UNDER = 0.01

// The share of a country the flag has to gain or lose for the dots to light.
export const DOT_SHARE = 0.1

// The share of a country the flag has to lose for the game to offer the video then.
export const LOSS_MOMENT_SHARE = 0.2

export const MOST_ALLIES = 3

type Held = {country: string, tiles: number, start: number, end: number}

export function storyOf(history: MapHistory, ground: Ground, flag: string): VideoStory | undefined {
    const [first, ...rest] = history.frames
    if (first === undefined) return undefined
    const owners = new Array<string>(ground.tiles + 1).fill("")
    const apply = (frame: MapFrame) => frame.tiles.forEach((tile, i) => {
        if (tile <= ground.tiles) owners[tile] = frame.owners[i]
    })
    apply(first)
    const start = heldIn(owners, ground, flag)
    rest.forEach(apply)
    const end = heldIn(owners, ground, flag)

    const sizes = new Map<string, number>()
    for (let tile = 1; tile <= ground.tiles; tile++) {
        const country = ground.groundOf(tile)
        if (country !== undefined) sizes.set(country, (sizes.get(country) ?? 0) + 1)
    }
    const held: Held[] = [...sizes].map(([country, tiles]) => ({country, tiles, start: start.get(country) ?? 0, end: end.get(country) ?? 0}))

    const [moved] = held.filter(({start, end}) => start !== end)
        .sort((a, b) => Math.abs(b.end - b.start) - Math.abs(a.end - a.start) || b.tiles - a.tiles || a.country.localeCompare(b.country))
    if (moved !== undefined && Math.abs(moved.end - moved.start) / moved.tiles >= QUIET_UNDER) {
        return {...moved, flag, kind: moved.end > moved.start ? "gain" : "loss"}
    }

    // A quiet day: where the flag holds the most, weighed by its share, so 62% of Poland comes before 1% of Russia
    // and before the whole of a one-tile seat.
    const [kept] = held.filter(({end}) => end > 0)
        .sort((a, b) => b.end * b.end / b.tiles - a.end * a.end / a.tiles || a.country.localeCompare(b.country))
    return kept && {...kept, flag, kind: "quiet"}
}

function heldIn(owners: readonly string[], ground: Ground, flag: string): Map<string, number> {
    const held = new Map<string, number>()
    owners.forEach((owner, tile) => {
        if (owner !== flag) return
        const country = ground.groundOf(tile)
        if (country !== undefined) held.set(country, (held.get(country) ?? 0) + 1)
    })
    return held
}

// The share of its country the flag gained or lost.
export function movedShare(story: VideoStory): number {
    return Math.abs(story.end - story.start) / story.tiles
}

// A whole percent, never more than it is, and never 0 for something.
export function percentOf(part: number, whole: number): number {
    if (part <= 0 || whole <= 0) return 0
    return Math.max(1, Math.floor(part / whole * 100))
}

export type Naming = {
    nameOf: (code: string) => string
    said: (name: string) => string
}

export function headlineOf(story: VideoStory, {nameOf, said}: Naming): string {
    const flag = said(nameOf(story.flag))
    const country = said(nameOf(story.country))
    switch (story.kind) {
        case "gain":
            return `${flag} took ${percentOf(story.end - story.start, story.tiles)}% of ${country}`.toUpperCase()
        case "loss":
            return `${flag} is losing ${country}. Help`.toUpperCase()
        case "quiet":
            return `${flag} holds ${percentOf(story.end, story.tiles)}% of ${country}`.toUpperCase()
    }
}

// The accounts that won the flag the most tiles of the story's country, most first, the maker left out. Only a change
// to the flag counts: never one who took from it.
export function alliesOf(history: RecentHistory, ground: Ground, story: VideoStory, maker: string | undefined): string[] {
    const won = new Map<string, number>()
    history.frames.forEach((frame, k) => {
        if (k === 0) return
        frame.tiles.forEach((tile, i) => {
            const account = frame.accounts[i]
            if (frame.owners[i] !== story.flag || account === "" || account === maker) return
            if (ground.groundOf(tile) === story.country) won.set(account, (won.get(account) ?? 0) + 1)
        })
    })
    return [...won].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([account]) => account)
}

// The names the end card shows: the maker's and its first allies', each only as the clips let a video show it.
export type Cast = {
    maker: string | undefined
    allies: string[]
}

export function castOf(maker: string | undefined, allies: readonly string[], shown: ReadonlyMap<string, string>): Cast {
    return {
        maker: maker === undefined ? undefined : shown.get(maker),
        allies: allies.flatMap((ally) => shown.get(ally) ?? []).slice(0, MOST_ALLIES),
    }
}
