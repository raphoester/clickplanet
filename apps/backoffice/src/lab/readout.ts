import {ArrivalCount, ArrivalSource} from "../gen/grpc/auth/v1/admin_pb.ts"

export type Platform = "tiktok" | "youtube"

export const PLATFORMS: Platform[] = ["tiktok", "youtube"]

export const CONTROL = "control"

// Arrivals are told by platform, and the lab's posts carry their experiment, from this day.
export const LAB_START = new Date("2026-11-20T00:00:00Z")

// A breakout is a post past its platform's line at 7 days.
export const BREAKOUT: Record<Platform, number> = {tiktok: 3000, youtube: 10_000}

const WEEK_HOURS = 7 * 24

// One post on one platform, read at its last row at 7 days or younger.
export type Post = {
    postId: string
    platform: Platform
    experiment: string
    publishedAt: Date
    // The UTC day it went out, as YYYY-MM-DD.
    day: string
    ageHours: number
    url: string
    views: number
    likes: number
    comments: number
    shares: number
    follows: number
    saves: number
    // TikTok's business numbers, which come 24 to 48 hours after a post: undefined until then.
    business?: Business
}

export type Business = {
    // The share still watching at 2 s; undefined for photos, which have no curve.
    heldAt2s: number | undefined
    toTheEnd: number
    averageWatchSeconds: number
}

type Analytics = {
    views?: number, likes?: number, comments?: number, shares?: number, follows?: number, saves?: number,
    completionRate?: number, profileViews?: number, igReelsAvgWatchTime?: number,
    impressionSources?: Record<string, number>, audienceCountries?: Record<string, number>,
    retentionCurve?: Record<string, number>,
}

type Document = {
    postId?: string, platform?: string, publishedAt?: string, ageHours?: number, url?: string,
    analytics?: {analytics?: Analytics}, metadata?: {record?: {post?: {experiment?: unknown}}},
}

// The lab's posts and the control's published since then, each at its last row at 7 days or younger. A post with no row
// that young, or no experiment, is not one.
export function postsOf(documents: string[], since: Date): Post[] {
    const read = new Map<string, Post>()
    for (const text of documents) {
        const post = postOf(JSON.parse(text) as Document)
        if (post === undefined || post.publishedAt < since || post.ageHours > WEEK_HOURS) continue
        const key = `${post.postId}/${post.platform}`
        const kept = read.get(key)
        if (kept === undefined || post.ageHours > kept.ageHours) read.set(key, post)
    }
    return [...read.values()].sort((a, b) => a.publishedAt.getTime() - b.publishedAt.getTime())
}

function postOf(document: Document): Post | undefined {
    const experiment = document.metadata?.record?.post?.experiment
    const platform = document.platform
    const metrics = document.analytics?.analytics
    if (typeof experiment !== "string" || experiment === "" || (platform !== "tiktok" && platform !== "youtube") ||
        document.postId === undefined || document.publishedAt === undefined || metrics === undefined) {
        return undefined
    }
    const publishedAt = new Date(document.publishedAt)
    return {
        postId: document.postId,
        platform,
        experiment,
        publishedAt,
        day: dayOf(publishedAt),
        ageHours: document.ageHours ?? 0,
        url: document.url ?? "",
        views: metrics.views ?? 0,
        likes: metrics.likes ?? 0,
        comments: metrics.comments ?? 0,
        shares: metrics.shares ?? 0,
        follows: metrics.follows ?? 0,
        saves: metrics.saves ?? 0,
        business: platform === "tiktok" ? businessOf(metrics) : undefined,
    }
}

// As the backend's Row.BusinessMetrics: any number only TikTok for Business gives.
function businessOf(metrics: Analytics): Business | undefined {
    const curve = metrics.retentionCurve ?? {}
    const live = (metrics.completionRate ?? 0) > 0 || (metrics.profileViews ?? 0) > 0 ||
        (metrics.igReelsAvgWatchTime ?? 0) > 0 || Object.keys(metrics.impressionSources ?? {}).length > 0 ||
        Object.keys(metrics.audienceCountries ?? {}).length > 0 || Object.keys(curve).length > 0
    if (!live) return undefined
    return {
        heldAt2s: curve["2"],
        toTheEnd: metrics.completionRate ?? 0,
        averageWatchSeconds: (metrics.igReelsAvgWatchTime ?? 0) / 1000,
    }
}

// Out under 7 days ago, so its reading still changes: a snapshot a day reads every post at 6 days and some hours.
export function youngOf(post: Post, now: Date): boolean {
    return now.getTime() - post.publishedAt.getTime() < WEEK_HOURS * 3_600_000
}

export function dayOf(at: Date): string {
    return at.toISOString().slice(0, 10)
}

// What an arm's posts add up to. A rate per 1,000 views is the posts' sum over their views, so a rare act still shows;
// a share is the median of the posts'.
export type Arm = {
    experiment: string
    posts: Post[]
    medianViews: number | undefined
    bestViews: number | undefined
    breakouts: number
    perThousand: {likes?: number, comments?: number, shares?: number, follows?: number, saves?: number}
    // TikTok's, over the posts whose business numbers came.
    business: {
        posts: number
        heldAt2s: number | undefined
        toTheEnd: number | undefined
        averageWatchSeconds: number | undefined
    }
}

export function armOf(experiment: string, platform: Platform, posts: Post[]): Arm {
    const views = posts.map(post => post.views)
    const measured = platform === "tiktok" ? posts.filter(post => post.business !== undefined) : posts
    return {
        experiment,
        posts,
        medianViews: medianOf(views),
        bestViews: views.length === 0 ? undefined : Math.max(...views),
        breakouts: posts.filter(post => post.views >= BREAKOUT[platform]).length,
        perThousand: {
            likes: perThousandOf(measured, post => post.likes),
            comments: perThousandOf(measured, post => post.comments),
            shares: perThousandOf(measured, post => post.shares),
            follows: perThousandOf(measured, post => post.follows),
            saves: perThousandOf(measured, post => post.saves),
        },
        business: {
            posts: posts.filter(post => post.business !== undefined).length,
            heldAt2s: medianOf(posts.flatMap(post => post.business?.heldAt2s ?? [])),
            toTheEnd: medianOf(posts.flatMap(post => post.business === undefined ? [] : [post.business.toTheEnd])),
            averageWatchSeconds: medianOf(posts.flatMap(post => post.business === undefined ? [] : [post.business.averageWatchSeconds])),
        },
    }
}

// An experiment on a platform, and the control's posts of the same UTC days there.
export type Reading = {
    first: string
    last: string
    experiment: Arm
    control: Arm
}

export function readingsOf(posts: Post[], platform: Platform): Reading[] {
    const on = posts.filter(post => post.platform === platform)
    const control = on.filter(post => post.experiment === CONTROL)
    const experiments = [...new Set(on.map(post => post.experiment))].filter(name => name !== CONTROL).sort()
    return experiments.map(name => {
        const its = on.filter(post => post.experiment === name)
        const days = its.map(post => post.day).sort()
        const first = days[0]
        const last = days[days.length - 1]
        return {
            first,
            last,
            experiment: armOf(name, platform, its),
            control: armOf(CONTROL, platform, control.filter(post => post.day >= first && post.day <= last)),
        }
    })
}

export function medianOf(values: number[]): number | undefined {
    if (values.length === 0) return undefined
    const sorted = [...values].sort((a, b) => a - b)
    const middle = Math.floor(sorted.length / 2)
    return sorted.length % 2 === 1 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2
}

function perThousandOf(posts: Post[], count: (post: Post) => number): number | undefined {
    const views = posts.reduce((sum, post) => sum + post.views, 0)
    if (views === 0) return undefined
    return 1000 * posts.reduce((sum, post) => sum + count(post), 0) / views
}

export type ArrivalDay = {day: string, tiktok: number, youtube: number}

// Each UTC day from since to today, the newest first.
export function arrivalDaysOf(counts: ArrivalCount[], since: Date, now: Date): ArrivalDay[] {
    const days: ArrivalDay[] = []
    for (let at = new Date(dayOf(since)); dayOf(at) <= dayOf(now); at = new Date(at.getTime() + 86_400_000)) {
        const day = dayOf(at)
        const of = (source: ArrivalSource) =>
            counts.filter(count => count.day === day && count.source === source).reduce((sum, count) => sum + count.accounts, 0)
        days.push({day, tiktok: of(ArrivalSource.TIKTOK), youtube: of(ArrivalSource.YOUTUBE)})
    }
    return days.reverse()
}

export function arrivalTotalsOf(days: ArrivalDay[]): {tiktok: number, youtube: number} {
    return days.reduce((sum, day) => ({tiktok: sum.tiktok + day.tiktok, youtube: sum.youtube + day.youtube}), {tiktok: 0, youtube: 0})
}
