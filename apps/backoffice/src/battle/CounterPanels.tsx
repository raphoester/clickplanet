import {Fact} from "../components/Fact.tsx"
import {clockOf} from "../components/time.ts"
import {wholeOf} from "../home/activity.ts"
import {
    labelValuesOf,
    lastMinuteOf,
    named,
    namesOf,
    nowOf,
    Sample,
    Select,
    shareUnderOf,
    sinceOf,
} from "./counters.ts"

const PERCENT = new Intl.NumberFormat("en", {style: "percent", maximumFractionDigits: 0})
const MB = 1024 * 1024

type Props = {samples: Sample[]}

// What the antibot reads now, by watchdog: a rule that reads many callers at the new pace is about to ban people.
export function AntibotPanel({samples}: Props) {
    const watchdogs = labelValuesOf(samples, "antibot_opinions_standing", "watchdog")
    const running = nowOf(samples, named("banned_callers"))

    return (
        <section className="panel" aria-label="Antibot">
            <div className="panel-head">
                <h2>Antibot</h2>
                <span className="muted">{running === undefined || Number.isNaN(running) ? "…" : wholeOf(running)} bans running</span>
            </div>
            {watchdogs.length === 0
                ? <p className="muted">No watchdog reads anybody yet.</p>
                : (
                    <table className="table">
                        <thead>
                            <tr>
                                <th scope="col">Watchdog</th>
                                <th scope="col" className="num">Suspect now</th>
                                <th scope="col" className="num">Certain now</th>
                                <th scope="col" className="num">Bans since {sinceLabelOf(samples)}</th>
                            </tr>
                        </thead>
                        <tbody>
                            {watchdogs.map(watchdog => (
                                <tr key={watchdog}>
                                    <td>{watchdog}</td>
                                    <td className="num">{countOf(nowOf(samples, named("antibot_opinions_standing", {watchdog, level: "suspect"})))}</td>
                                    <td className="num">{countOf(nowOf(samples, named("antibot_opinions_standing", {watchdog, level: "certain"})))}</td>
                                    <td className="num">{wholeOf(Math.round(sinceOf(samples, named("ban_flags", {watchdog}))))}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
        </section>
    )
}

// The boxes: how many go out, how many are caught, and how fast.
export function BoxPanel({samples}: Props) {
    const caughtFast = shareUnderOf(samples, "bonus_catch_seconds", 1)
    return (
        <section className="panel" aria-label="Boxes">
            <div className="panel-head">
                <h2>Boxes</h2>
                <span className="muted">a minute, and since {sinceLabelOf(samples)}</span>
            </div>
            <dl className="facts">
                <Rate label="Offered" samples={samples} select={named("bonus_offers_total")}/>
                <Rate label="Caught" samples={samples} select={named("bonus_claims_total", {outcome: "granted"})}/>
                <Rate label="Lapsed" samples={samples} select={named("bonus_lapsed_total")}/>
                <Rate label="Claims refused" samples={samples} select={named("bonus_claims_total", {outcome: "refused"})}/>
                <Rate label="Another's box" samples={samples} select={named("bonus_claims_foreign_total")}/>
                <Fact label="Caught within 1 s">{caughtFast === undefined ? "…" : PERCENT.format(caughtFast)}</Fact>
            </dl>
        </section>
    )
}

// How the server holds up: the process's CPU and memory, and what the event bus dropped.
export function ServerPanel({samples}: Props) {
    const cpu = lastMinuteOf(samples, named("process_cpu_seconds_total"))
    const memory = nowOf(samples, named("process_resident_memory_bytes"))
    const goroutines = nowOf(samples, named("go_goroutines"))
    return (
        <section className="panel" aria-label="Server">
            <div className="panel-head">
                <h2>Server</h2>
            </div>
            <dl className="facts">
                <Fact label="CPU">{cpu === undefined ? "…" : `${PERCENT.format(cpu / 60)} of a core`}</Fact>
                <Fact label="Memory">{memory === undefined ? "…" : `${wholeOf(Math.round(memory / MB))} MB`}</Fact>
                <Fact label="Goroutines">{countOf(goroutines)}</Fact>
                <Rate label="Events dropped" samples={samples} select={named("events_dropped_total")}/>
            </dl>
        </section>
    )
}

// Every counter and gauge the server keeps, its labels summed: for what no panel shows.
export function EveryCounter({samples}: Props) {
    const names = namesOf(samples)
    return (
        <details className="chart-table">
            <summary>Every counter</summary>
            <table className="table">
                <thead>
                    <tr>
                        <th scope="col">Name</th>
                        <th scope="col" className="num">A minute</th>
                        <th scope="col" className="num">Since {sinceLabelOf(samples)}</th>
                        <th scope="col" className="num">Now</th>
                    </tr>
                </thead>
                <tbody>
                    {names.map(({name, kind}) => (
                        <tr key={name}>
                            <td className="mono">{name}</td>
                            <td className="num">{kind === "counter" ? rateOf(lastMinuteOf(samples, named(name))) : ""}</td>
                            <td className="num">{kind === "counter" ? wholeOf(Math.round(sinceOf(samples, named(name)))) : ""}</td>
                            <td className="num">{kind === "gauge" ? countOf(nowOf(samples, named(name))) : ""}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </details>
    )
}

function Rate({label, samples, select}: {label: string, samples: Sample[], select: Select}) {
    return (
        <Fact label={label}>
            {rateOf(lastMinuteOf(samples, select))}, <span className="muted">{wholeOf(Math.round(sinceOf(samples, select)))}</span>
        </Fact>
    )
}

function sinceLabelOf(samples: Sample[]): string {
    return samples.length === 0 ? "now" : clockOf(new Date(samples[0].at))
}

function rateOf(rate: number | undefined): string {
    return rate === undefined ? "…" : wholeOf(Math.round(rate))
}

function countOf(value: number | undefined): string {
    return value === undefined || Number.isNaN(value) ? "…" : wholeOf(Math.round(value))
}
