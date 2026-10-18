import {useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {timeOf} from "../components/time.ts"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {replay, replayFileOf, RETENTION_HOURS} from "./actions.ts"
import {download, SavedFile} from "./download.ts"

const HOUR_MS = 60 * 60 * 1000
const SPANS = [1, 6, 12, 24, RETENTION_HOURS]

type Saved = {name: string, events: number, since: Date | undefined, until: Date | undefined}

// The map as it was, and every act on it since: what the clip generator plays back. A read, so it asks nothing.
export function ReplayPanel({planet, save = download}: {planet: PlanetAdmin, save?: (file: SavedFile) => void}) {
    const [hours, setHours] = useState(12)
    const [ending, setEnding] = useState("")
    const [saved, start] = useAction<Saved>(wordsOf)

    const fetchReplay = () => start(async () => {
        // A datetime-local is the operator's own time; empty is now, which the server reads as no end.
        const until = ending === "" ? undefined : new Date(ending)
        const since = new Date((until ?? new Date()).getTime() - hours * HOUR_MS)
        const answer = await replay(planet, {since, until})
        const file = replayFileOf(answer)
        save(file)
        return {name: file.name, events: answer.events.length, since: answer.since?.toDate(), until: answer.until?.toDate()}
    })

    return (
        <section className="panel" aria-label="Download a replay">
            <h2>Download a replay</h2>
            <p className="hint">
                The file <code>clips fetch</code> saves, for the frontend's <code>npm run clip</code>. The ledger keeps
                {" "}{RETENTION_HOURS} hours: an older start opens on the oldest act it has. A day is a few MB.
            </p>
            <div className="fields">
                <label className="picker">
                    <span className="muted">Span</span>
                    <select value={hours} onChange={event => setHours(Number(event.target.value))}>
                        {SPANS.map(span => <option key={span} value={span}>{countOf(span, "hour")}</option>)}
                    </select>
                </label>
                <label className="field">
                    <span className="muted">Ending (empty: now)</span>
                    <input type="datetime-local" value={ending} onChange={event => setEnding(event.target.value)}/>
                </label>
            </div>
            <div className="action">
                <button type="button" className="button" disabled={saved.step === "running"} onClick={fetchReplay}>
                    {saved.step === "running" ? "Fetching…" : "Download"}
                </button>
                {saved.step === "refused" && <p className="action-refused" role="alert">{saved.words}</p>}
                {saved.step === "done" && (
                    <p className="action-said" role="status">
                        Saved {saved.value.name}: {countOf(saved.value.events, "event")}
                        {saved.value.since !== undefined && saved.value.until !== undefined
                            && `, ${timeOf(saved.value.since)} to ${timeOf(saved.value.until)}`}.
                    </p>
                )}
            </div>
        </section>
    )
}
