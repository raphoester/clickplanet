import {Timestamp} from "@bufbuild/protobuf"
import {useCallback, useState} from "react"
import {AuthAdmin, PublishingAdmin} from "../clients.ts"
import {documentsOf} from "../clips/actions.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {useCall} from "../useCall.ts"
import {
    Arm, ArrivalDay, arrivalDaysOf, BREAKOUT, dayOf, LAB_START, Platform, PLATFORMS, Post, postsOf, Reading, readingsOf,
} from "./readout.ts"

const NAMES: Record<Platform, string> = {tiktok: "TikTok", youtube: "YouTube"}

const SHOWN_DAYS = 14

const clock = () => new Date()

// Each experiment of the lab on each platform against the control of the same days, and the arrivals of each platform.
export function LabPage({publishing, auth, now = clock}: {publishing: PublishingAdmin, auth: AuthAdmin, now?: () => Date}) {
    const [opened] = useState(now)
    const today = dayOf(opened)
    const [since, setSince] = useState(() => dayOf(new Date(Math.max(LAB_START.getTime(), opened.getTime() - SHOWN_DAYS * 86_400_000))))

    const readPosts = useCallback(async (signal: AbortSignal) => {
        const from = new Date(`${since}T00:00:00Z`)
        return postsOf(await documentsOf(publishing, {since: from, until: undefined}, signal), from)
    }, [publishing, since])
    const [posts, readAgain] = useCall(readPosts)

    const countArrivals = useCallback(async (signal: AbortSignal) => {
        const from = new Date(`${since}T00:00:00Z`)
        const answer = await auth.countArrivals({since: Timestamp.fromDate(from)}, {signal})
        return arrivalDaysOf(answer.counts, from, opened)
    }, [auth, since, opened])
    const [arrivals, countAgain] = useCall(countArrivals)

    return (
        <section aria-label="Lab" className="lab">
            <div className="fields">
                <label className="field">
                    <span className="muted">Posts since (UTC)</span>
                    <input
                        type="date"
                        value={since}
                        min={dayOf(LAB_START)}
                        max={today}
                        onChange={event => {
                            if (event.target.value >= dayOf(LAB_START) && event.target.value <= today) setSince(event.target.value)
                        }}
                    />
                </label>
            </div>
            {posts.state === "loading" && <p className="muted">Reading the snapshots…</p>}
            {posts.state === "failed" && <FailureNotice failure={posts} onRetry={readAgain}/>}
            {posts.state === "done" && PLATFORMS.map(platform => (
                <PlatformReadout key={platform} platform={platform} posts={posts.value}/>
            ))}
            <section className="panel" aria-label="Arrivals">
                <h2>Arrivals</h2>
                {arrivals.state === "loading" && <p className="muted">Counting…</p>}
                {arrivals.state === "failed" && <FailureNotice failure={arrivals} onRetry={countAgain}/>}
                {arrivals.state === "done" && <Arrivals days={arrivals.value}/>}
            </section>
        </section>
    )
}

function PlatformReadout({platform, posts}: {platform: Platform, posts: Post[]}) {
    const readings = readingsOf(posts, platform)
    const shown = posts.filter(post => post.platform === platform)
    const tiktok = platform === "tiktok"
    return (
        <section className="panel" aria-label={NAMES[platform]}>
            <h2>{NAMES[platform]}</h2>
            {readings.length === 0
                ? <p className="muted">No lab post.</p>
                : (
                    <div className="scroll-x">
                        <table className="table lab-table">
                            <thead>
                                <tr>
                                    <th scope="col">Experiment</th>
                                    <th scope="col">Days</th>
                                    <th scope="col" className="num">Posts</th>
                                    <th scope="col" className="num">Median views</th>
                                    <th scope="col" className="num">Best</th>
                                    <th scope="col" className="num" title={`${count(BREAKOUT[platform])} views or more at 7 days`}>
                                        Breakouts
                                    </th>
                                    {tiktok && <>
                                        <th scope="col" className="num">Held at 2 s</th>
                                        <th scope="col" className="num">To the end</th>
                                        <th scope="col" className="num">Average watch</th>
                                        <th scope="col" className="num">Comments /1k</th>
                                        <th scope="col" className="num">Shares /1k</th>
                                        <th scope="col" className="num">Follows /1k</th>
                                        <th scope="col" className="num">Saves /1k</th>
                                    </>}
                                    {!tiktok && <th scope="col" className="num">Likes /1k</th>}
                                </tr>
                            </thead>
                            {readings.map(reading => (
                                <tbody key={reading.experiment.experiment}>
                                    <ArmRow arm={reading.experiment} reading={reading} tiktok={tiktok}/>
                                    <ArmRow arm={reading.control} tiktok={tiktok}/>
                                </tbody>
                            ))}
                        </table>
                    </div>
                )}
            {shown.length > 0 && <PostList posts={shown} tiktok={tiktok}/>}
        </section>
    )
}

// The experiment's row holds the days both rows read; the control's is under it.
function ArmRow({arm, reading, tiktok}: {arm: Arm, reading?: Reading, tiktok: boolean}) {
    const control = reading === undefined
    const waiting = arm.posts.length - arm.business.posts
    return (
        <tr className={control ? "lab-control" : undefined}>
            <th scope="row">{arm.experiment}</th>
            {reading !== undefined && <td rowSpan={2} className="lab-days">{daysOf(reading.first, reading.last)}</td>}
            <td className="num">
                {arm.posts.length}
                {tiktok && waiting > 0 && arm.posts.length > 0 && <span className="muted"> ({waiting} too young)</span>}
            </td>
            <td className="num">{count(arm.medianViews)}</td>
            <td className="num">{count(arm.bestViews)}</td>
            <td className="num">{arm.breakouts > 0 ? <strong>{arm.breakouts}</strong> : arm.posts.length > 0 ? 0 : "—"}</td>
            {tiktok && <>
                <td className="num">{percent(arm.business.heldAt2s)}</td>
                <td className="num">{percent(arm.business.toTheEnd)}</td>
                <td className="num">{secondsOf(arm.business.averageWatchSeconds)}</td>
                <td className="num">{rate(arm.perThousand.comments)}</td>
                <td className="num">{rate(arm.perThousand.shares)}</td>
                <td className="num">{rate(arm.perThousand.follows)}</td>
                <td className="num">{rate(arm.perThousand.saves)}</td>
            </>}
            {!tiktok && <td className="num">{rate(arm.perThousand.likes)}</td>}
        </tr>
    )
}

function PostList({posts, tiktok}: {posts: Post[], tiktok: boolean}) {
    return (
        <details>
            <summary>The {posts.length} posts</summary>
            <div className="scroll-x">
                <table className="table">
                    <thead>
                        <tr>
                            <th scope="col">Out (UTC)</th>
                            <th scope="col">Experiment</th>
                            <th scope="col" className="num">Read at</th>
                            <th scope="col" className="num">Views</th>
                            {tiktok && <>
                                <th scope="col" className="num">Held at 2 s</th>
                                <th scope="col" className="num">To the end</th>
                                <th scope="col" className="num">Average watch</th>
                            </>}
                            <th scope="col">Post</th>
                        </tr>
                    </thead>
                    <tbody>
                        {[...posts].reverse().map(post => (
                            <tr key={post.postId}>
                                <td>{post.publishedAt.toISOString().slice(0, 16).replace("T", " ")}</td>
                                <td>{post.experiment}</td>
                                <td className="num">{Math.round(post.ageHours)} h</td>
                                <td className="num">{count(post.views)}</td>
                                {tiktok && <>
                                    <td className="num">{percent(post.business?.heldAt2s)}</td>
                                    <td className="num">{percent(post.business?.toTheEnd)}</td>
                                    <td className="num">{secondsOf(post.business?.averageWatchSeconds)}</td>
                                </>}
                                <td>{post.url === "" ? "" : <a href={post.url} target="_blank" rel="noreferrer">open</a>}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </details>
    )
}

function Arrivals({days}: {days: ArrivalDay[]}) {
    return (
        <table className="table">
            <thead>
                <tr>
                    <th scope="col">Day (UTC)</th>
                    <th scope="col" className="num">TikTok (/t)</th>
                    <th scope="col" className="num">YouTube (/y)</th>
                </tr>
            </thead>
            <tbody>
                {days.map(day => (
                    <tr key={day.day}>
                        <td>{daysOf(day.day, day.day)}</td>
                        <td className="num">{day.tiktok}</td>
                        <td className="num">{day.youtube}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

const DAY = new Intl.DateTimeFormat("en-GB", {day: "numeric", month: "short", timeZone: "UTC"})

// 20 Nov, 20–23 Nov, 30 Nov – 2 Dec.
function daysOf(first: string, last: string): string {
    const from = DAY.format(new Date(`${first}T00:00:00Z`))
    if (first === last) return from
    const to = DAY.format(new Date(`${last}T00:00:00Z`))
    return first.slice(0, 7) === last.slice(0, 7) ? `${from.split(" ")[0]}–${to}` : `${from} – ${to}`
}

function count(value: number | undefined): string {
    return value === undefined ? "—" : Math.round(value).toLocaleString("en-US")
}

function percent(value: number | undefined): string {
    return value === undefined ? "—" : `${Math.round(value * 100)}%`
}

function secondsOf(value: number | undefined): string {
    return value === undefined ? "—" : `${value.toFixed(1)} s`
}

function rate(value: number | undefined): string {
    return value === undefined ? "—" : value.toFixed(1)
}
