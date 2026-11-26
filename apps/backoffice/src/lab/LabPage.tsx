import {Timestamp} from "@bufbuild/protobuf"
import {useCallback, useState} from "react"
import {AuthAdmin, PublishingAdmin} from "../clients.ts"
import {documentsOf} from "../clips/actions.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {useCall} from "../useCall.ts"
import {ArrivalsChart} from "./ArrivalsChart.tsx"
import {colorOf, experimentsOf, logTicksOf} from "./chart.ts"
import {count, daysOf, momentOf, percent, rate, secondsOf} from "./format.ts"
import {Key, Legend} from "./Legend.tsx"
import {
    Arm, ArrivalDay, arrivalDaysOf, BREAKOUT, CONTROL, dayOf, LAB_START, Platform, PLATFORMS, Post, postsOf, Reading,
    readingsOf, youngOf,
} from "./readout.ts"
import {SignsChart} from "./SignsChart.tsx"
import {ViewsByDayChart} from "./ViewsByDayChart.tsx"
import {ViewsChart} from "./ViewsChart.tsx"

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
                <PlatformReadout key={platform} platform={platform} posts={posts.value} since={since} now={opened}
                                 experiments={experimentsOf(posts.value)}/>
            ))}
            <section className="panel" aria-label="Arrivals">
                <h2>Arrivals</h2>
                {arrivals.state === "loading" && <p className="muted">Counting…</p>}
                {arrivals.state === "failed" && <FailureNotice failure={arrivals} onRetry={countAgain}/>}
                {arrivals.state === "done" && <>
                    <ArrivalsChart days={arrivals.value} today={today}/>
                    <details className="chart-table">
                        <summary>Table</summary>
                        <Arrivals days={arrivals.value}/>
                    </details>
                </>}
            </section>
        </section>
    )
}

type ReadoutProps = {platform: Platform, posts: Post[], since: string, now: Date, experiments: string[]}

function PlatformReadout({platform, posts, since, now, experiments}: ReadoutProps) {
    const [hovered, setHovered] = useState<string | undefined>(undefined)
    const [picked, setPicked] = useState<string | undefined>(undefined)
    const readings = readingsOf(posts, platform)
    const theirs = posts.filter(post => post.platform === platform)
    const tiktok = platform === "tiktok"
    const paint = (experiment: string) => colorOf(experiment, experiments)
    const ticks = logTicksOf(Math.max(BREAKOUT[platform], ...theirs.map(post => post.views)))
    const pointed = theirs.find(post => post.postId === (hovered ?? picked))
    const keys: Key[] = [
        ...readings.map(({experiment}) => ({
            label: experiment.experiment, mark: "dot" as const, color: paint(experiment.experiment),
        })),
        {label: CONTROL, mark: "dot", color: paint(CONTROL)},
        {label: "Median", mark: "median"},
        {label: "Under 7 days", mark: "young"},
        {label: "Breakout", mark: "breakout"},
    ]
    const charts = {
        ticks, breakout: BREAKOUT[platform], now, colorOf: paint, shown: pointed?.postId, onHover: setHovered, onPick: setPicked,
    }

    return (
        <section className="panel" aria-label={NAMES[platform]}>
            <h2>{NAMES[platform]}</h2>
            {readings.length === 0
                ? <p className="muted">No lab post.</p>
                : <>
                    <Legend keys={keys}/>
                    <p className="readout" aria-live="polite">
                        {pointed === undefined
                            ? <>{theirs.length} posts, {theirs.filter(post => youngOf(post, now)).length} under 7 days</>
                            : <>{pointed.experiment}, {momentOf(pointed.publishedAt)}: <b>{count(pointed.views)}</b> views
                                at {Math.round(pointed.ageHours)}&nbsp;h
                                {pointed.url !== "" && <> · <a href={pointed.url} target="_blank" rel="noreferrer">open</a></>}</>}
                    </p>
                    <ViewsChart readings={readings} {...charts}/>
                    <ViewsByDayChart posts={theirs} since={since} today={dayOf(now)} {...charts}/>
                    <SignsChart platform={platform} readings={readings} colorOf={paint}/>
                    <details className="chart-table">
                        <summary>Table</summary>
                        <ReadingTable platform={platform} readings={readings}/>
                    </details>
                </>}
            {theirs.length > 0 && <PostList posts={theirs} tiktok={tiktok}/>}
        </section>
    )
}

function ReadingTable({platform, readings}: {platform: Platform, readings: Reading[]}) {
    const tiktok = platform === "tiktok"
    return (
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
