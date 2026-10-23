import {Timestamp} from "@bufbuild/protobuf"
import {ReactNode, useCallback, useEffect, useMemo, useState} from "react"
import {Clients, CLIENTS, SupportAdmin} from "../clients.ts"
import {clockOf} from "../components/time.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetActivityResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {PageId} from "../navigation.ts"
import {Call, useCall} from "../useCall.ts"
import {ActivityCharts} from "./ActivityCharts.tsx"
import {durationOf, FIRST_SPAN, slotsOf, Span, SPANS, stepOf, totalsOf, wholeOf} from "./activity.ts"
import {CountryTable} from "./CountryTable.tsx"
import {Grid} from "./heat.ts"
import {Heatmap} from "./Heatmap.tsx"
import {PlayingNow} from "./PlayingNow.tsx"

export const REFRESH_MS = 60_000
const DAY_MS = 24 * 3600_000
// The most ListReports and ListNotes answer at once.
const PAGE = 500

type Inbox = {openReports: number, notesToday: number}

type Props = {clients?: Clients, onGo?: (page: PageId) => void}

export function HomePage({clients = CLIENTS, onGo}: Props) {
    const [span, setSpan] = useState<Span>(FIRST_SPAN)

    const readActivity = useCallback(
        (signal: AbortSignal) => clients.planet.getActivity({span: durationOf(span)}, {signal}),
        [clients, span],
    )
    const readRoster = useCallback(
        (signal: AbortSignal) => clients.player.getRoster({}, {signal}).then(answer => answer.entries),
        [clients],
    )
    const readInbox = useCallback((signal: AbortSignal) => inboxOf(clients.support, signal), [clients])

    const [activityCall, readActivityAgain] = useCall(readActivity)
    const [rosterCall, readRosterAgain] = useCall(readRoster)
    const [inboxCall, readInboxAgain] = useCall(readInbox)

    const reload = useCallback(() => {
        readActivityAgain()
        readRosterAgain()
        readInboxAgain()
    }, [readActivityAgain, readRosterAgain, readInboxAgain])

    useEffect(() => {
        const timer = setInterval(reload, REFRESH_MS)
        return () => clearInterval(timer)
    }, [reload])

    const activity = useLast(activityCall)
    const roster = useLast(rosterCall)
    const inbox = useLast(inboxCall)

    const slots = useMemo(() => activity === undefined ? [] : slotsOf(activity), [activity])
    const grid = useMemo(() => activity === undefined ? undefined : gridOf(activity), [activity])
    const reading = [activityCall, rosterCall, inboxCall].some(call => call.state === "loading")

    return (
        <div className={reading ? "home home-reading" : "home"} aria-busy={reading}>
            <div className="toolbar">
                <div className="segmented" role="group" aria-label="Span">
                    {SPANS.map(item => (
                        <button key={item.hours} type="button" aria-pressed={span.hours === item.hours}
                                onClick={() => setSpan(item)}>
                            {item.label}
                        </button>
                    ))}
                </div>
                <span className="toolbar-end">
                    {activity?.until !== undefined && <span className="muted">Updated {clockOf(activity.until.toDate())}</span>}
                    <button type="button" className="button" onClick={reload}>Reload</button>
                </span>
            </div>

            <ul className="tiles">
                <Tile label="Playing now" value={roster?.length}
                      sub={roster === undefined ? undefined : `${wholeOf(roster.filter(entry => !entry.guest).length)} signed in`}/>
                <Tile label="Map tabs open" value={activity?.streams}
                      sub={activity === undefined ? undefined : `${wholeOf(activity.watchers)} networks or accounts`}/>
                <Tile label="Clicks" value={activity === undefined ? undefined : totalsOf(slots).clicks}
                      sub={`last ${span.label}`}/>
                <Tile label="Players" value={activity?.players} sub={`took a tile, last ${span.label}`}/>
                <Tile label="Open reports" value={inbox?.openReports} more={inbox?.openReports === PAGE}
                      onGo={onGo === undefined ? undefined : () => onGo("reports")}/>
                <Tile label="Notes today" value={inbox?.notesToday} more={inbox?.notesToday === PAGE}
                      onGo={onGo === undefined ? undefined : () => onGo("notes")}/>
            </ul>
            {rosterCall.state === "failed" && <FailureNotice failure={rosterCall} onRetry={readRosterAgain}/>}
            {inboxCall.state === "failed" && <FailureNotice failure={inboxCall} onRetry={readInboxAgain}/>}

            {activityCall.state === "failed" && <FailureNotice failure={activityCall} onRetry={readActivityAgain}/>}
            {activity === undefined && activityCall.state === "loading" && <p className="muted">Loading…</p>}
            {activity !== undefined && (
                <>
                    <ActivityCharts slots={slots} step={stepOf(activity)} span={span} players={activity.players}/>
                    {grid !== undefined && <Heatmap grid={grid}/>}
                    <CountryTable countries={activity.countries} mapTiles={activity.mapTiles} span={span}/>
                </>
            )}

            {roster !== undefined && <PlayingNow roster={roster}/>}
        </div>
    )
}

type TileProps = {label: string, value: number | undefined, sub?: string, more?: boolean, onGo?: () => void}

function Tile({label, value, sub, more = false, onGo}: TileProps) {
    const body: ReactNode = (
        <>
            <span className="tile-label">{label}</span>
            <span className="tile-value">{value === undefined ? "…" : `${wholeOf(value)}${more ? "+" : ""}`}</span>
            {sub !== undefined && <span className="tile-sub">{sub}</span>}
        </>
    )

    return (
        <li className="tile">
            {onGo === undefined ? <div className="tile-body">{body}</div>
                : <button type="button" className="tile-body tile-link" onClick={onGo}>{body}</button>}
        </li>
    )
}

// The last answer stays shown, dimmed, while the next one is read.
function useLast<T>(call: Call<T>): T | undefined {
    const [last, setLast] = useState<T | undefined>(undefined)

    useEffect(() => {
        if (call.state === "done") setLast(call.value)
    }, [call])

    return call.state === "done" ? call.value : last
}

async function inboxOf(support: SupportAdmin, signal: AbortSignal): Promise<Inbox> {
    const since = Timestamp.fromDate(new Date(Date.now() - DAY_MS))
    const [reports, notes] = await Promise.all([
        support.listReports({openOnly: true, limit: PAGE}, {signal}),
        support.listNotes({since, limit: PAGE}, {signal}),
    ])
    return {openReports: reports.reports.length, notesToday: notes.notes.length}
}

function gridOf(activity: GetActivityResponse): Grid | undefined {
    const heatmap = activity.heatmap
    if (heatmap === undefined || heatmap.columns === 0) return undefined
    return {columns: heatmap.columns, rows: heatmap.rows, land: heatmap.land, changes: heatmap.changes}
}
