import {Appearance, ClipLink, Platform} from "../backends/clips.ts"
import {TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../links.ts"

export const CLIPS_SHOWN_STORAGE_KEY = "clickplanet-clips-shown"

export const APPEARANCES_READ_GAP_MS = 10 * 60_000

const SHOWN_KEPT = 50

export function readDue(lastReadAt: number | undefined, now: number): boolean {
    return lastReadAt === undefined || now - lastReadAt >= APPEARANCES_READ_GAP_MS
}

export type WatchLink = {
    platform: Platform
    label: string
    url: string
}

export function watchLinksOf(appearance: Appearance): WatchLink[] {
    const platforms = new Set<Platform>()
    return appearance.links.flatMap((link) => {
        if (platforms.has(link.platform)) return []
        platforms.add(link.platform)
        return [watchLinkOf(link)]
    })
}

function watchLinkOf({platform, url}: ClipLink): WatchLink {
    switch (platform) {
        case "tiktok":
            return {platform, label: "Watch on TikTok", url: url || TIKTOK_PROFILE}
        case "youtube":
            return {platform, label: "Watch on YouTube", url: url || YOUTUBE_CHANNEL}
    }
}

export function appearanceToShow(appearances: readonly Appearance[], shown: readonly string[]): Appearance | undefined {
    return appearances
        .filter((appearance) => !shown.includes(appearance.postId) && watchLinksOf(appearance).length > 0)
        .reduce<Appearance | undefined>(
            (newest, appearance) => !newest || appearance.scheduledAt > newest.scheduledAt ? appearance : newest, undefined)
}

// Once one clip was shown, the older ones of the same read are no news either.
export function shownAfter(shown: readonly string[], seen: readonly Appearance[]): string[] {
    const ids = [...new Set(seen.map((appearance) => appearance.postId))]
    return [...ids, ...shown.filter((id) => !ids.includes(id))].slice(0, SHOWN_KEPT)
}

export function parseShownClips(raw: string | null): string[] {
    if (raw === null) return []

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return []
    }
    if (!Array.isArray(stored)) return []

    return stored.filter((id): id is string => typeof id === "string")
}
