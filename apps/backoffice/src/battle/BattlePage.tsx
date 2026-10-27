import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {useCallback, useMemo, useRef, useState} from "react"
import {AntibotAdmin, Clients, CLIENTS, PlayerAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {clockOf} from "../components/time.ts"
import {Conviction} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {wholeOf} from "../home/activity.ts"
import {Ledger, readOn, slotsOf, windowOf} from "../home/ledger.ts"
import {Tile} from "../home/HomePage.tsx"
import {PlayingNow} from "../home/PlayingNow.tsx"
import {ANTIBOT_OFF, namesOf, Named} from "../players/actions.ts"
import {BanList} from "./BanList.tsx"
import {AntibotPanel, BoxPanel, EveryCounter} from "./CounterPanels.tsx"
import {
    keptSamples,
    lastMinuteOf,
    MINUTE,
    named,
    perSlotOf,
    reasonLabelOf,
    reasonsOf,
    refused,
    Sample,
    sampleOf,
} from "./counters.ts"
import {MinuteChart, Series, SLOTS} from "./MinuteChart.tsx"
import {RacePanel} from "./RacePanel.tsx"
import {Polled, usePolling} from "./usePolling.ts"

export const REFRESH_MS = 15_000
const PERCENT = new Intl.NumberFormat("en", {style: "percent", maximumFractionDigits: 1})

export type Span = {hours: number, label: string, every: number}

const SPANS: Span[] = [
    {hours: 0.5, label: "30 minutes", every: 5},
    {hours: 1, label: "1 hour", every: 10},
    {hours: 2, label: "2 hours", every: 15},
    {hours: 3, label: "3 hours", every: 30},
]

const FIRST_SPAN = SPANS[2]

type Props = {clients?: Clients}

export function BattlePage({clients = CLIENTS}: Props) {
    const [span, setSpan] = useState<Span>(FIRST_SPAN)
    const [hovered, setHovered] = useState<number | undefined>(undefined)
    const keptLedger = useRef<Ledger | undefined>(undefined)
    const planetSamples = useRef<Sample[]>([])
    const antibotSamples = useRef<Sample[]>([])

    const readLedger = useCallback(async (signal: AbortSignal) => {
        keptLedger.current = await readOn(clients.planet, keptLedger.current, windowOf(new Date(), span.hours, MINUTE), signal)
        return keptLedger.current
    }, [clients, span])
    const readCounters = useCallback(async (signal: AbortSignal) => {
        const answer = await clients.planet.getMetrics({}, {signal})
        planetSamples.current = keptSamples(planetSamples.current, sampleOf(answer), span.hours * 3600_000)
        return planetSamples.current
    }, [clients, span])
    const readWatchdogs = useCallback(async (signal: AbortSignal) => {
        const answer = await clients.antibot.getMetrics({}, {signal})
        antibotSamples.current = keptSamples(antibotSamples.current, sampleOf(answer), span.hours * 3600_000)
        return antibotSamples.current
    }, [clients, span])
    const readBans = useCallback(
        (signal: AbortSignal) => bansOf(clients.antibot, clients.player, span, signal),
        [clients, span],
    )
    const readRace = useCallback(
        (signal: AbortSignal) => clients.seasons.getRace({}, {signal}).then(answer => answer.race),
        [clients],
    )
    const readRoster = useCallback(
        (signal: AbortSignal) => clients.player.getRoster({}, {signal}).then(answer => answer.entries),
        [clients],
    )

    const [ledger, readLedgerAgain] = usePolling(readLedger, REFRESH_MS)
    const [counters, readCountersAgain] = usePolling(readCounters, REFRESH_MS)
    const [watchdogs, readWatchdogsAgain] = usePolling(readWatchdogs, REFRESH_MS)
    const [bans, readBansAgain] = usePolling(readBans, REFRESH_MS)
    const [race, readRaceAgain] = usePolling(readRace, REFRESH_MS)
    const [roster, readRosterAgain] = usePolling(readRoster, REFRESH_MS)

    const reload = () => {
        readLedgerAgain()
        readCountersAgain()
        readWatchdogsAgain()
        readBansAgain()
        readRaceAgain()
        readRosterAgain()
    }

    const frame = ledger.value?.window
    const slots = useMemo(() => ledger.value === undefined ? [] : slotsOf(ledger.value, ledger.value.window), [ledger.value])
    const kept = useMemo(() => counters.value ?? [], [counters.value])
    const reasons = useMemo(() => reasonsOf(kept), [kept])
    // Clicks hold the first slot: a reason never wears its color.
    const refusals: Series[] = useMemo(() => frame === undefined ? [] : reasons.map((reason, i) => ({
        key: reason, label: reasonLabelOf(reason), slot: Math.min(i + 2, SLOTS),
        values: perSlotOf(kept, frame, named(reason)),
    })), [kept, reasons, frame])

    const settled = slots.length >= 2 ? slots[slots.length - 2] : undefined
    const refusedNow = lastMinuteOf(kept, refused)
    const acceptedNow = lastMinuteOf(kept, named("clicks_total"))
    const share = refusedNow === undefined || acceptedNow === undefined || refusedNow + acceptedNow === 0
        ? undefined : refusedNow / (refusedNow + acceptedNow)
    const shown = hovered === undefined ? undefined : slots[hovered]
    const reading = [ledger, counters, watchdogs, bans, race, roster].some(polled => polled.reading)

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
                    {frame !== undefined && <span className="muted">Updated {clockOf(frame.until)}</span>}
                    <button type="button" className="button" onClick={reload}>Reload</button>
                </span>
            </div>

            <ul className="tiles">
                <Tile label="Playing now" value={roster.value?.length}
                      sub={roster.value === undefined ? undefined : `${wholeOf(roster.value.filter(entry => !entry.guest).length)} signed in`}/>
                <Tile label="Clicks a minute" value={settled?.clicks}
                      sub={settled === undefined ? undefined : `at ${clockOf(settled.start)}`}/>
                <Tile label="Refused a minute" value={refusedNow === undefined ? undefined : Math.round(refusedNow)}
                      sub={share === undefined ? undefined : `${PERCENT.format(share)} of clicks`}/>
                <Tile label="Bans" value={bans.value?.off === false ? bans.value.convictions.length : undefined}
                      sub={`last ${span.label}`}/>
            </ul>

            <Failed polled={bans} onRetry={readBansAgain}/>
            {bans.value?.off === true && <p className="notice">{ANTIBOT_OFF}</p>}
            {bans.value?.off === false && (
                <BanList antibot={clients.antibot} convictions={bans.value.convictions}
                         authors={bans.value.authors} unnamed={bans.value.unnamed}/>
            )}

            <Failed polled={ledger} onRetry={readLedgerAgain}/>
            <Failed polled={counters} onRetry={readCountersAgain}/>
            {frame !== undefined && (
                <section className="panel" aria-label="Clicks">
                    <p className="readout" aria-live="polite">
                        {shown === undefined
                            ? <>Last {span.label}: <b>{wholeOf(slots.reduce((sum, slot) => sum + slot.clicks, 0))}</b> clicks</>
                            : <>{clockOf(shown.start)}{shown.filling ? " (so far)" : ""}: <b>{wholeOf(shown.clicks)}</b> clicks
                                {refusals.map(line => (
                                    <span key={line.key}>, {line.label} <b>{amountOf(line.values[hovered ?? 0])}</b></span>
                                ))}</>}
                    </p>
                    <MinuteChart title="Clicks a minute" starts={slots.map(slot => slot.start)} every={span.every}
                                 series={[{key: "clicks", label: "clicks", slot: 1, values: slots.map(slot => slot.clicks)}]}
                                 filling hovered={hovered} onHover={setHovered}/>
                    <MinuteChart title="Refused a minute" starts={slots.map(slot => slot.start)} every={span.every}
                                 series={refusals} filling hovered={hovered} onHover={setHovered}/>
                    <details className="chart-table">
                        <summary>Table</summary>
                        <table className="table">
                            <thead>
                                <tr>
                                    <th scope="col">Minute</th>
                                    <th scope="col" className="num">Clicks</th>
                                    {refusals.map(line => <th key={line.key} scope="col" className="num">{line.label}</th>)}
                                </tr>
                            </thead>
                            <tbody>
                                {slots.map((slot, i) => (
                                    <tr key={slot.start.getTime()}>
                                        <td>{clockOf(slot.start)}</td>
                                        <td className="num">{wholeOf(slot.clicks)}</td>
                                        {refusals.map(line => <td key={line.key} className="num">{amountOf(line.values[i])}</td>)}
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </details>
                </section>
            )}

            <Failed polled={race} onRetry={readRaceAgain}/>
            {race.value !== undefined && <RacePanel race={race.value} now={new Date()}/>}

            <Failed polled={watchdogs} onRetry={readWatchdogsAgain}/>
            {watchdogs.value !== undefined && <AntibotPanel samples={watchdogs.value}/>}
            {counters.value !== undefined && <BoxPanel samples={kept}/>}
            {counters.value !== undefined && <EveryCounter title="Every planet counter" samples={kept}/>}
            {watchdogs.value !== undefined && <EveryCounter title="Every antibot counter" samples={watchdogs.value}/>}

            <Failed polled={roster} onRetry={readRosterAgain}/>
            {roster.value !== undefined && <PlayingNow roster={roster.value}/>}
        </div>
    )
}

type Bans = {convictions: Conviction[], off: boolean} & Named

async function bansOf(antibot: AntibotAdmin, player: PlayerAdmin, span: Span, signal: AbortSignal): Promise<Bans> {
    const since = Timestamp.fromDate(new Date(Date.now() - span.hours * 3600_000))
    let convictions: Conviction[]
    try {
        convictions = (await antibot.listConvictions({since}, {signal})).convictions
    } catch (err) {
        if (ConnectError.from(err).code !== Code.Unimplemented) throw err
        return {convictions: [], off: true, authors: new Map(), unnamed: undefined}
    }
    const accounts = convictions.flatMap(conviction => [conviction.accountId, conviction.alongsideAccountId])
    return {convictions, off: false, ...await namesOf(player, accounts, signal)}
}

function Failed<T>({polled, onRetry}: {polled: Polled<T>, onRetry: () => void}) {
    return polled.failure === undefined ? null : <FailureNotice failure={polled.failure} onRetry={onRetry}/>
}

function amountOf(value: number | undefined): string {
    return value === undefined ? "–" : wholeOf(Math.round(value))
}
