import {Timestamp} from "@bufbuild/protobuf"
import {ReactNode, useCallback, useEffect, useMemo, useState} from "react"
import {LONG_CALL_MS} from "../api.ts"
import {Clients, CLIENTS, PlanetAdmin, SupportAdmin} from "../clients.ts"
import {clockOf} from "../components/time.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {PageId} from "../navigation.ts"
import {Call, useCall} from "../useCall.ts"
import {ActivityCharts} from "./ActivityCharts.tsx"
import {FIRST_SPAN, Span, SPANS, totalsOf, wholeOf} from "./activity.ts"
import {CountryTable} from "./CountryTable.tsx"
import {Heatmap} from "./Heatmap.tsx"
import {changesOf, countriesOf, playersOf, slotsOf, windowOf} from "./ledger.ts"
import {loadPlaces, Places} from "./places.ts"
import {PlayingNow} from "./PlayingNow.tsx"

export const REFRESH_MS = 60_000
const DAY_MS = 24 * 3600_000
// The most ListReports and ListNotes answer at once.
const PAGE = 500

type Props = {clients?: Clients, onGo?: (page: PageId) => void, places?: () => Promise<Places>}

export function HomePage({clients = CLIENTS, onGo, places: readPlaces = loadPlaces}: Props) {
    const [span, setSpan] = useState<Span>(FIRST_SPAN)

    const readLedger = useCallback((signal: AbortSignal) => ledgerOf(clients.planet, span, signal), [clients, span])
    const readMap = useCallback((signal: AbortSignal) => mapOf(clients.planet, signal), [clients])
    const readRoster = useCallback(
        (signal: AbortSignal) => clients.player.getRoster({}, {signal}).then(answer => answer.entries),
        [clients],
    )
    const readInbox = useCallback((signal: AbortSignal) => inboxOf(clients.support, signal), [clients])
    const readTiles = useCallback(() => readPlaces(), [readPlaces])

    const [ledgerCall, readLedgerAgain] = useCall(readLedger)
    const [mapCall, readMapAgain] = useCall(readMap)
    const [rosterCall, readRosterAgain] = useCall(readRoster)
    const [inboxCall, readInboxAgain] = useCall(readInbox)
    const [placesCall, readPlacesAgain] = useCall(readTiles)

    const reload = useCallback(() => {
        readLedgerAgain()
        readMapAgain()
        readRosterAgain()
        readInboxAgain()
    }, [readLedgerAgain, readMapAgain, readRosterAgain, readInboxAgain])

    useEffect(() => {
        const timer = setInterval(reload, REFRESH_MS)
        return () => clearInterval(timer)
    }, [reload])

    const ledger = useLast(ledgerCall)
    const map = useLast(mapCall)
    const roster = useLast(rosterCall)
    const inbox = useLast(inboxCall)
    const places = placesCall.state === "done" ? placesCall.value : undefined

    const slots = useMemo(() => ledger === undefined ? [] : slotsOf(ledger.answer, ledger.window), [ledger])
    const players = useMemo(() => ledger === undefined ? 0 : playersOf(ledger.answer), [ledger])
    const countries = useMemo(
        () => ledger === undefined || map === undefined ? [] : countriesOf(ledger.answer, map.territories),
        [ledger, map],
    )
    const changes = useMemo(
        () => ledger === undefined || places === undefined ? undefined : changesOf(ledger.answer, places.latitudes.length),
        [ledger, places],
    )
    const reading = [ledgerCall, mapCall, rosterCall, inboxCall].some(call => call.state === "loading")
    const shownSpan = ledger?.span ?? span

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
                    {ledger !== undefined && <span className="muted">Updated {clockOf(ledger.window.until)}</span>}
                    <button type="button" className="button" onClick={reload}>Reload</button>
                </span>
            </div>

            <ul className="tiles">
                <Tile label="Playing now" value={roster?.length}
                      sub={roster === undefined ? undefined : `${wholeOf(roster.filter(entry => !entry.guest).length)} signed in`}/>
                <Tile label="Clicks" value={ledger === undefined ? undefined : totalsOf(slots).clicks}
                      sub={`last ${shownSpan.label}`}/>
                <Tile label="Players" value={ledger === undefined ? undefined : players}
                      sub={`took a tile, last ${shownSpan.label}`}/>
                <Tile label="Open reports" value={inbox?.openReports} more={inbox?.openReports === PAGE}
                      onGo={onGo === undefined ? undefined : () => onGo("reports")}/>
                <Tile label="Notes today" value={inbox?.notesToday} more={inbox?.notesToday === PAGE}
                      onGo={onGo === undefined ? undefined : () => onGo("notes")}/>
            </ul>
            {mapCall.state === "failed" && <FailureNotice failure={mapCall} onRetry={readMapAgain}/>}
            {rosterCall.state === "failed" && <FailureNotice failure={rosterCall} onRetry={readRosterAgain}/>}
            {inboxCall.state === "failed" && <FailureNotice failure={inboxCall} onRetry={readInboxAgain}/>}

            {ledgerCall.state === "failed" && <FailureNotice failure={ledgerCall} onRetry={readLedgerAgain}/>}
            {ledger === undefined && ledgerCall.state === "loading" && <p className="muted">Loading…</p>}
            {ledger !== undefined && (
                <>
                    <ActivityCharts slots={slots} step={ledger.window.step} span={ledger.span} players={players}/>
                    {placesCall.state === "failed" && <FailureNotice failure={placesCall} onRetry={readPlacesAgain}/>}
                    {places !== undefined && changes !== undefined && <Heatmap places={places} changes={changes}/>}
                    {map !== undefined && <CountryTable countries={countries} mapTiles={map.tiles} span={ledger.span}/>}
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

async function ledgerOf(planet: PlanetAdmin, span: Span, signal: AbortSignal) {
    const window = windowOf(new Date(), span.hours, span.step)
    const answer = await planet.getLedger(
        {since: Timestamp.fromDate(window.since), until: Timestamp.fromDate(window.until)},
        {signal, timeoutMs: LONG_CALL_MS},
    )
    return {answer, window, span}
}

async function mapOf(planet: PlanetAdmin, signal: AbortSignal) {
    const {territories, tiles} = await planet.getTerritories({}, {signal})
    return {territories, tiles}
}

async function inboxOf(support: SupportAdmin, signal: AbortSignal) {
    const since = Timestamp.fromDate(new Date(Date.now() - DAY_MS))
    const [reports, notes] = await Promise.all([
        support.listReports({openOnly: true, limit: PAGE}, {signal}),
        support.listNotes({since, limit: PAGE}, {signal}),
    ])
    return {openReports: reports.reports.length, notesToday: notes.notes.length}
}
