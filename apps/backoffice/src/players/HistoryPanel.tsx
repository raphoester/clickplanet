import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {Call} from "../useCall.ts"
import {actorWords, HISTORY_LIMIT, Moderated} from "./history.ts"

type Props = {history: Call<Moderated[]>, onReload: () => void}

// What was done to the account in the chat and to its name, newest first, with the reason and who did it: Claude's
// actions told apart from the operator's. Kept 90 days, past the chat's own retention.
export function HistoryPanel({history, onReload}: Props) {
    return (
        <section className="panel" aria-label="Moderation history">
            <div className="panel-head">
                <h2>Moderation history</h2>
                <button type="button" className="button-small" onClick={onReload}>Read again</button>
            </div>
            {history.state === "loading" && <p className="muted">Loading…</p>}
            {history.state === "failed" && <FailureNotice failure={history} onRetry={onReload}/>}
            {history.state === "done" && history.value.length === 0 && (
                <p className="muted">No mute, removal or rename in the last 90 days.</p>
            )}
            {history.state === "done" && history.value.length > 0 && (
                <ol className="history">
                    {history.value.map(each => <Line key={each.id} line={each}/>)}
                </ol>
            )}
            {history.state === "done" && history.value.length >= HISTORY_LIMIT && (
                <p className="muted">The latest {HISTORY_LIMIT} of each kind.</p>
            )}
        </section>
    )
}

function Line({line}: {line: Moderated}) {
    const by = actorWords(line.by)
    return (
        <li className="history-line">
            <div className="history-head">
                <When at={line.at}/>
                <strong>{line.what}</strong>
                <span
                    className={line.by.kind === "app" ? "badge badge-app" : "badge"}
                    title={line.by.kind === "operator" ? line.by.accountId : undefined}
                >
                    {by}
                </span>
            </div>
            {line.said !== undefined && "text" in line.said && <blockquote className="history-said">{line.said.text}</blockquote>}
            {line.said !== undefined && "gifUrl" in line.said && (
                <a className="history-said" href={line.said.gifUrl} target="_blank" rel="noreferrer">The GIF</a>
            )}
            {line.reason === ""
                ? <p className="muted">No reason given.</p>
                : <p><span className="muted">Reason: </span><span>{line.reason}</span></p>}
        </li>
    )
}
