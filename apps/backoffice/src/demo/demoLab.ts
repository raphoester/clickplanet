import {Timestamp} from "@bufbuild/protobuf"
import {ArrivalCount, ArrivalSource} from "../gen/grpc/auth/v1/admin_pb.ts"
import {StatsRow} from "../gen/grpc/publishing/v1/admin_pb.ts"

const HOUR = 3_600_000
const DAY = 24 * HOUR

type MadeUp = {postId: string, platform: "tiktok" | "youtube", experiment: string, at: number, views: number}

// Ten days of the lab, made up: on TikTok far invasion, the carousel and the control each day; on YouTube the control
// half the time; the war report on both for the last two days, too young for TikTok's signs. A snapshot a day at
// 08:17 UTC, each post's views climbing to where it settles. One day has no arrival from either platform.
export function labOf(now: number, roll: () => number): {rows: StatsRow[], arrivals: ArrivalCount[]} {
    const today = Math.floor(now / DAY) * DAY
    const posts: MadeUp[] = []
    for (let day = 10; day >= 0; day--) {
        const start = today - day * DAY
        for (const [hour, experiment] of [[14, "far-invasion"], [16, "carousel"], [18.75, "control"]] as const) {
            const views = experiment === "far-invasion" && day === 2 ? 4200 : 450 + Math.floor(roll() * 500)
            posts.push({postId: `lab-${day}-${hour}`, platform: "tiktok", experiment, at: start + hour * HOUR, views})
        }
        for (const [hour, experiment] of [[9, "control"], [12, "far-invasion"], [15, "control"], [17, "carousel"], [20, "control"], [22, "far-invasion"]] as const) {
            const views = experiment === "far-invasion" && day === 4 && hour === 22
                ? 15_000
                : 700 + Math.floor(roll() * (experiment === "carousel" ? 600 : 2500))
            posts.push({postId: `lab-${day}-${hour}`, platform: "youtube", experiment, at: start + hour * HOUR, views})
        }
        if (day <= 1) {
            posts.push({postId: `lab-${day}-11`, platform: "tiktok", experiment: "war-report", at: start + 11 * HOUR, views: 200 + Math.floor(roll() * 300)})
            posts.push({postId: `lab-${day}-19`, platform: "youtube", experiment: "war-report", at: start + 19 * HOUR, views: 250 + Math.floor(roll() * 400)})
        }
    }

    const rows: StatsRow[] = []
    for (let day = 10; day >= 0; day--) {
        const taken = today - day * DAY + 8 * HOUR + 17 * 60_000
        if (taken > now) continue
        for (const post of posts.filter(post => post.at < taken)) {
            const ageHours = (taken - post.at) / HOUR
            rows.push(rowOf(post, taken, ageHours))
        }
    }

    const arrivals = Array.from({length: 11}, (_, back) => {
        const day = new Date(today - back * DAY).toISOString().slice(0, 10)
        const unknown = new ArrivalCount({day, source: ArrivalSource.UNKNOWN, accounts: 30 + Math.floor(roll() * 20)})
        if (back === 3) return [unknown]
        return [
            new ArrivalCount({day, source: ArrivalSource.TIKTOK, accounts: Math.floor(roll() * 3)}),
            new ArrivalCount({day, source: ArrivalSource.YOUTUBE, accounts: 3 + Math.floor(roll() * 4)}),
            unknown,
        ]
    }).flat().reverse()

    return {rows, arrivals}
}

function rowOf(post: MadeUp, taken: number, ageHours: number): StatsRow {
    const views = Math.round(post.views * (1 - Math.exp(-ageHours / 30)))
    const photos = post.experiment === "carousel"
    const business = post.platform === "tiktok" && ageHours > 36 ? {
        completionRate: photos ? 0.18 : 0.15 + (post.views % 7) / 100,
        igReelsAvgWatchTime: photos ? 6800 : 5000 + post.views % 900,
        impressionSources: {forYou: 0.96},
        retentionCurve: photos ? {} : {"0": 1, "1": 0.82, "2": 0.6 + (post.views % 9) / 100, "3": 0.5},
    } : {}
    const metrics = {
        views, likes: Math.round(views * 0.012), comments: Math.round(views * 0.0015), shares: Math.round(views * 0.0004),
        follows: post.platform === "tiktok" && ageHours > 36 ? Math.round(views * 0.0006) : 0,
        saves: Math.round(views * (photos ? 0.004 : 0.0015)), ...business,
    }
    const url = post.platform === "tiktok"
        ? `https://www.tiktok.com/@clickplanetlol/video/${post.postId}`
        : `https://www.youtube.com/watch?v=${post.postId}`
    return new StatsRow({
        takenAt: Timestamp.fromDate(new Date(taken)), postId: post.postId, platform: post.platform,
        document: JSON.stringify({
            postId: post.postId, platform: post.platform, takenAt: new Date(taken).toISOString(),
            publishedAt: new Date(post.at).toISOString(), ageHours: Math.round(ageHours * 100) / 100, url,
            answer: {httpStatus: 200, status: "published"}, analytics: {analytics: metrics},
            metadata: {record: {post: {experiment: post.experiment}, clip: {format: photos ? "carousel" : "globe-replay"}}},
        }),
    })
}
