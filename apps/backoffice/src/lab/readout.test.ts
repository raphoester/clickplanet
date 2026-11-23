import {describe, expect, it} from "vitest"
import {ArrivalCount, ArrivalSource} from "../gen/grpc/auth/v1/admin_pb.ts"
import {armOf, arrivalDaysOf, medianOf, Post, postsOf, readingsOf} from "./readout.ts"

const SINCE = new Date("2026-11-20T00:00:00Z")

type Row = {
    postId?: string, platform?: string, publishedAt?: string, ageHours?: number, experiment?: string,
    metrics?: Record<string, unknown>, teaser?: boolean,
}

function documentOf(row: Row): string {
    return JSON.stringify({
        postId: row.postId ?? "post-1", platform: row.platform ?? "tiktok",
        publishedAt: row.publishedAt ?? "2026-11-21T16:00:00Z", ageHours: row.ageHours ?? 40,
        url: "https://www.tiktok.com/@clickplanetlol/video/1",
        analytics: {analytics: row.metrics ?? {views: 500}},
        metadata: row.teaser
            ? {teaserBattle: "battle-1"}
            : {record: row.experiment === undefined ? {} : {post: {experiment: row.experiment}}},
    })
}

function post(experiment: string, day: string, views: number, more: Partial<Post> = {}): Post {
    return {
        postId: `${experiment}-${day}-${views}`, platform: "youtube", experiment, publishedAt: new Date(`${day}T12:00:00Z`), day,
        ageHours: 48, url: "", views, likes: 0, comments: 0, shares: 0, follows: 0, saves: 0, ...more,
    }
}

describe("postsOf", () => {
    it("reads each post of the lab or the control at its last row at 7 days or younger", () => {
        const posts = postsOf([
            documentOf({experiment: "carousel", ageHours: 20, metrics: {views: 100}}),
            documentOf({experiment: "carousel", ageHours: 165, metrics: {views: 900}}),
            documentOf({experiment: "carousel", ageHours: 189, metrics: {views: 950}}),
            documentOf({experiment: "carousel", platform: "youtube", ageHours: 44, metrics: {views: 1200}}),
        ], SINCE)

        expect(posts.map(each => [each.platform, each.ageHours, each.views])).toEqual([
            ["tiktok", 165, 900],
            ["youtube", 44, 1200],
        ])
        expect(posts[0].day).toBe("2026-11-21")
    })

    it("leaves out a post with no experiment, a teaser, and a post out before the window", () => {
        const posts = postsOf([
            documentOf({postId: "old-clip"}),
            documentOf({postId: "teaser", teaser: true}),
            documentOf({postId: "before", experiment: "control", publishedAt: "2026-11-19T23:00:00Z"}),
            documentOf({postId: "kept", experiment: "control"}),
        ], SINCE)

        expect(posts.map(each => each.postId)).toEqual(["kept"])
    })

    it("reads TikTok's business numbers once they came, and no hold at 2 s for photos", () => {
        const [young, video, photos] = postsOf([
            documentOf({postId: "young", experiment: "control", metrics: {views: 300, completionRate: 0, retentionCurve: {}}}),
            documentOf({
                postId: "video", experiment: "control",
                metrics: {views: 541, completionRate: 0.2159, igReelsAvgWatchTime: 6590, retentionCurve: {"1": 0.83, "2": 0.65}},
            }),
            documentOf({
                postId: "photos", experiment: "carousel",
                metrics: {views: 524, completionRate: 0.1609, igReelsAvgWatchTime: 7390, retentionCurve: {}},
            }),
        ].map((document, i) => document.replace("2026-11-21T16:00:00Z", `2026-11-21T1${i}:00:00Z`)), SINCE)

        expect(young.business).toBeUndefined()
        expect(video.business).toEqual({heldAt2s: 0.65, toTheEnd: 0.2159, averageWatchSeconds: 6.59})
        expect(photos.business).toEqual({heldAt2s: undefined, toTheEnd: 0.1609, averageWatchSeconds: 7.39})
    })
})

describe("armOf", () => {
    it("counts the median and the best views, and the posts past the platform's line", () => {
        const youtube = armOf("far-invasion", "youtube", [post("far-invasion", "2026-11-20", 9_999), post("far-invasion", "2026-11-21", 10_000)])
        expect([youtube.medianViews, youtube.bestViews, youtube.breakouts]).toEqual([9_999.5, 10_000, 1])

        const tiktok = armOf("far-invasion", "tiktok", [post("far-invasion", "2026-11-20", 3_000, {platform: "tiktok"})])
        expect(tiktok.breakouts).toBe(1)
    })

    it("rates each act per 1,000 views over the posts' sums, so a rare one still shows", () => {
        const arm = armOf("control", "youtube", [
            post("control", "2026-11-20", 1_000, {likes: 6}),
            post("control", "2026-11-21", 3_000, {likes: 26}),
        ])

        expect(arm.perThousand.likes).toBe(8)
    })

    it("reads TikTok's signs over the posts whose business numbers came", () => {
        const business = (held: number | undefined) => ({heldAt2s: held, toTheEnd: 0.16, averageWatchSeconds: 5.3})
        const arm = armOf("control", "tiktok", [
            post("control", "2026-11-20", 600, {platform: "tiktok", follows: 1, business: business(0.5)}),
            post("control", "2026-11-21", 400, {platform: "tiktok", follows: 0, business: business(0.75)}),
            post("control", "2026-11-22", 9_000, {platform: "tiktok", follows: 5}),
        ])

        expect(arm.business).toEqual({posts: 2, heldAt2s: 0.625, toTheEnd: 0.16, averageWatchSeconds: 5.3})
        expect(arm.perThousand.follows).toBe(1)
        expect(arm.medianViews).toBe(600)
    })

    it("has no median of no post", () => {
        expect(medianOf([])).toBeUndefined()
        expect(armOf("control", "youtube", []).perThousand.likes).toBeUndefined()
    })
})

describe("readingsOf", () => {
    it("sets each experiment against the control of its own days on its platform", () => {
        const readings = readingsOf([
            post("control", "2026-11-19", 1),
            post("control", "2026-11-20", 2),
            post("far-invasion", "2026-11-20", 10),
            post("carousel", "2026-11-21", 20),
            post("control", "2026-11-21", 3),
            post("far-invasion", "2026-11-22", 30),
            post("control", "2026-11-23", 4),
            post("carousel", "2026-11-21", 5, {platform: "tiktok"}),
        ], "youtube")

        expect(readings.map(reading => [reading.experiment.experiment, reading.first, reading.last])).toEqual([
            ["carousel", "2026-11-21", "2026-11-21"],
            ["far-invasion", "2026-11-20", "2026-11-22"],
        ])
        expect(readings[0].control.posts.map(each => each.views)).toEqual([3])
        expect(readings[1].control.posts.map(each => each.views)).toEqual([2, 3])
        expect(readings[0].experiment.posts).toHaveLength(1)
    })
})

describe("arrivalDaysOf", () => {
    it("gives each day from the window's start to today, the newest first, with none as 0", () => {
        const days = arrivalDaysOf([
            new ArrivalCount({day: "2026-11-20", source: ArrivalSource.TIKTOK, accounts: 4}),
            new ArrivalCount({day: "2026-11-20", source: ArrivalSource.UNKNOWN, accounts: 40}),
            new ArrivalCount({day: "2026-11-22", source: ArrivalSource.YOUTUBE, accounts: 7}),
        ], SINCE, new Date("2026-11-22T10:00:00Z"))

        expect(days).toEqual([
            {day: "2026-11-22", tiktok: 0, youtube: 7},
            {day: "2026-11-21", tiktok: 0, youtube: 0},
            {day: "2026-11-20", tiktok: 4, youtube: 0},
        ])
    })
})
