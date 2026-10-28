import {useCallback, useEffect, useState} from "react"
import {GamemasterAdmin, PlanetAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {clockOf, timeOf} from "../components/time.ts"
import {useCall} from "../useCall.ts"
import {Block, Instant, LANES, placeOf, Timeline, timelineOf, untilOf, upcoming} from "./timeline.ts"

const HOUR = 3600 * 1000
const DAY = 24 * HOUR

const RANGES = [
    {label: "1 day", span: DAY},
    {label: "3 days", span: 3 * DAY},
    {label: "1 week", span: 7 * DAY},
    {label: "4 weeks", span: 28 * DAY},
]

export function TimelinePanel({gamemaster, planet}: {gamemaster: GamemasterAdmin, planet: PlanetAdmin}) {
    const read = useCallback(async (signal: AbortSignal) => {
        const [schedule, seasons, rules] = await Promise.all([
            gamemaster.getSchedule({}, {signal}), gamemaster.getSeasons({}, {signal}), planet.getRules({}, {signal}),
        ])
        return timelineOf({schedule, seasons, planet: rules})
    }, [gamemaster, planet])
    const [state, again] = useCall(read)
    const [span, setSpan] = useState(3 * DAY)
    const now = useNow()

    return (
        <section aria-label="Timeline" className="timeline">
            <div className="timeline-tools">
                <div className="segmented" role="group" aria-label="Range">
                    {RANGES.map(range => (
                        <button key={range.label} type="button" aria-pressed={span === range.span} onClick={() => setSpan(range.span)}>
                            {range.label}
                        </button>
                    ))}
                </div>
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>
            {state.state === "loading" && <p className="muted">Loading…</p>}
            {state.state === "failed" && <FailureNotice failure={state} onRetry={again}/>}
            {state.state === "done" && (
                <>
                    <Lanes timeline={state.value} from={new Date(now.getTime() - span / 12)} to={new Date(now.getTime() + span)} now={now}/>
                    <Chronology instants={upcoming(state.value.instants, now).filter(instant => instant.at.getTime() <= now.getTime() + span)} now={now}/>
                </>
            )}
        </section>
    )
}

function useNow(): Date {
    const [now, setNow] = useState(() => new Date())
    useEffect(() => {
        const timer = setInterval(() => setNow(new Date()), 30 * 1000)
        return () => clearInterval(timer)
    }, [])
    return now
}

function Lanes({timeline, from, to, now}: {timeline: Timeline, from: Date, to: Date, now: Date}) {
    const nowAt = placeOf({startsAt: now, endsAt: to}, from, to)?.left
    return (
        <div className="lanes" role="table" aria-label="Blocks">
            <Ticks from={from} to={to}/>
            {LANES.map(({lane, label}) => (
                <div key={lane} className="lane" role="row">
                    <span className="lane-label muted" role="rowheader">{label}</span>
                    <div className="lane-track" role="cell">
                        {timeline.blocks.filter(block => block.lane === lane).map(block => (
                            <BlockBar key={`${block.label}-${block.startsAt.getTime()}`} block={block} from={from} to={to}/>
                        ))}
                        {nowAt !== undefined && <span className="lane-now" style={{left: `${nowAt}%`}} aria-hidden="true"/>}
                    </div>
                </div>
            ))}
        </div>
    )
}

function BlockBar({block, from, to}: {block: Block, from: Date, to: Date}) {
    const place = placeOf(block, from, to)
    if (place === undefined) return null
    const title = `${block.label}${block.detail === "" ? "" : ` — ${block.detail}`}\n${timeOf(block.startsAt)} → ${timeOf(block.endsAt)}`
    return (
        <span className={`block block-${block.tone}`} style={{left: `${place.left}%`, width: `${place.width}%`}} title={title}>
            {block.label}
        </span>
    )
}

// A tick at each midnight, or at each week's start when midnights would crowd the track.
function Ticks({from, to}: {from: Date, to: Date}) {
    const step = to.getTime() - from.getTime() > 10 * DAY ? 7 * DAY : DAY
    const first = new Date(from)
    first.setHours(24, 0, 0, 0)
    const ticks: Date[] = []
    for (let at = first.getTime(); at < to.getTime(); at += step) ticks.push(new Date(at))
    return (
        <div className="lane lane-ticks" aria-hidden="true">
            <span className="lane-label"/>
            <div className="lane-track">
                {ticks.map(tick => (
                    <span key={tick.getTime()} className="tick muted" style={{left: `${placeOf({startsAt: tick, endsAt: to}, from, to)?.left ?? 0}%`}}>
                        {tick.toLocaleDateString(undefined, {weekday: "short", day: "numeric"})}
                    </span>
                ))}
            </div>
        </div>
    )
}

function Chronology({instants, now}: {instants: Instant[], now: Date}) {
    if (instants.length === 0) return <p className="muted">Nothing happens in this range.</p>
    return (
        <ol className="chronology" aria-label="What happens next">
            {instants.map((instant, i) => (
                <li key={instant.at.getTime()} className="instant" aria-current={i === 0 ? "true" : undefined}>
                    <div className="instant-when">
                        <strong>{instant.at.toLocaleDateString(undefined, {weekday: "short", day: "numeric", month: "short"})} {clockOf(instant.at)}</strong>
                        <span className="muted">{i === 0 ? `Next, ${untilOf(instant.at, now)}` : untilOf(instant.at, now)}</span>
                    </div>
                    <ul className="instant-what">
                        {instant.happenings.map(happening => (
                            <li key={happening.what}><span className={`dot block-${happening.tone}`} aria-hidden="true"/>{happening.what}</li>
                        ))}
                    </ul>
                </li>
            ))}
        </ol>
    )
}
