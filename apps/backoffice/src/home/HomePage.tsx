import {Timestamp} from "@bufbuild/protobuf"
import {ReactNode, useCallback, useEffect, useMemo, useRef, useState} from "react"
import {LONG_CALL_MS} from "../api.ts"
import {loadPlaces, Places} from "../boards/places.ts"
import {Clients, CLIENTS, PlanetAdmin, SupportAdmin} from "../clients.ts"
import {clockOf} from "../components/time.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {LedgerChange} from "../gen/grpc/planet/v1/ledger_pb.ts"
import {PageId} from "../navigation.ts"
import {Call, useCall} from "../useCall.ts"
import {usePageVisible} from "../usePageVisible.ts"
import {ActivityCharts} from "./ActivityCharts.tsx"
import {FIRST_SPAN, Span, SPANS, totalsOf, wholeOf} from "./activity.ts"
import {CountryTable} from "./CountryTable.tsx"
import {Heatmap} from "./Heatmap.tsx"
import {addsTo, boardOfPages, caughtUp, changesOf, countriesOf, Ledger, playersOf, sinceOf, slotsOf, windowOf} from "./ledger.ts"
import {PlayingNow} from "./PlayingNow.tsx"

export const REFRESH_MS = 60_000
const DAY_MS = 24 * 3600_000
// The most ListReports and ListNotes answer at once.
const PAGE = 500

// A ledger's read that the world's board changed under, again and again, gives up.
const READS = 3

// places: a board's places, by the name an answer gives.
type Props = {clients?: Clients, onGo?: (page: PageId) => void, places?: (board: string) => Promise<Places>}

export function HomePage({clients = CLIENTS, onGo, places: readPlaces = loadPlaces}: Props) {
    const [span, setSpan] = useState<Span>(FIRST_SPAN)
    const visible = usePageVisible()

    const kept = useRef<Ledger | undefined>(undefined)
    const readLedger = useCallback(async (signal: AbortSignal) => {
        const read = await ledgerOf(clients.planet, span, kept.current, signal)
        if (!signal.aborted) kept.current = read
        return read
    }, [clients, span])
    const readMap = useCallback((signal: AbortSignal) => mapOf(clients.planet, signal), [clients])
    const readRoster = useCallback(
        (signal: AbortSignal) => clients.player.getRoster({}, {signal}).then(answer => answer.entries),
        [clients],
    )
    const readInbox = useCallback((signal: AbortSignal) => inboxOf(clients.support, signal), [clients])

    const [ledgerCall, readLedgerAgain] = useCall(readLedger)
    const [mapCall, readMapAgain] = useCall(readMap)
    const [rosterCall, readRosterAgain] = useCall(readRoster)
    const [inboxCall, readInboxAgain] = useCall(readInbox)

    const ledger = useLast(ledgerCall)
    const map = useLast(mapCall)
    const roster = useLast(rosterCall)
    const inbox = useLast(inboxCall)

    // The tiles are placed on the board the ledger names, read once that board is known.
    const board = ledger?.board
    const readTiles = useCallback(
        () => board === undefined ? new Promise<Places>(() => {}) : readPlaces(board),
        [readPlaces, board],
    )
    const [placesCall, readPlacesAgain] = useCall(readTiles)

    const reload = useCallback(() => {
        readLedgerAgain()
        readMapAgain()
        readRosterAgain()
        readInboxAgain()
    }, [readLedgerAgain, readMapAgain, readRosterAgain, readInboxAgain])

    const reading = [ledgerCall, mapCall, rosterCall, inboxCall].some(call => call.state === "loading")

    const endedAt = useRef(0)
    useEffect(() => {
        if (!reading) endedAt.current = Date.now()
    }, [reading])

    // The next read is a minute after the last one ended, and waits while the page is hidden.
    useEffect(() => {
        if (reading || !visible) return
        const timer = setTimeout(reload, Math.max(0, endedAt.current + REFRESH_MS - Date.now()))
        return () => clearTimeout(timer)
    }, [reading, visible, reload])

    const places = placesCall.state === "done" && placesCall.value.board.name === board ? placesCall.value : undefined

    const slots = useMemo(() => ledger === undefined ? [] : slotsOf(ledger, ledger.window), [ledger])
    const players = useMemo(() => ledger === undefined ? 0 : playersOf(ledger), [ledger])
    const countries = useMemo(
        () => ledger === undefined || map === undefined ? [] : countriesOf(ledger, map.territories),
        [ledger, map],
    )
    const changes = useMemo(
        () => ledger === undefined || places === undefined ? undefined : changesOf(ledger, places.latitudes.length),
        [ledger, places],
    )
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
                    <button type="button" className="button" onClick={reload} disabled={reading}>Reload</button>
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

// The span's changes: what was kept, and what was read since. A read that finds the world on another board than the
// one kept, or a board switched between two of its pages, reads the span whole again, on the board the world plays now.
async function ledgerOf(planet: PlanetAdmin, span: Span, kept: Ledger | undefined, signal: AbortSignal) {
    const window = windowOf(new Date(), span.hours, span.step)
    let since = sinceOf(kept, window)
    for (let reads = 1; ; reads++) {
        const {changes, board} = await pagesOf(planet, since, window.until, signal)
        if (board !== undefined && addsTo(kept, window, since, board)) return {...caughtUp(kept, window, since, changes, board), span}
        if (reads === READS) throw new Error("the world's board changed while its ledger was read: reload")
        since = window.since
    }
}

// Every page of the ledger from `since`, one after the other, and the board they all name.
async function pagesOf(planet: PlanetAdmin, since: Date, until: Date, signal: AbortSignal) {
    const changes: LedgerChange[] = []
    const boards: string[] = []
    let page = ""
    do {
        const answer = await planet.getLedger(
            {since: Timestamp.fromDate(since), until: Timestamp.fromDate(until), page},
            {signal, timeoutMs: LONG_CALL_MS},
        )
        for (const change of answer.changes) changes.push(change)
        boards.push(answer.board)
        page = answer.nextPage
    } while (page !== "")
    return {changes, board: boardOfPages(boards)}
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
