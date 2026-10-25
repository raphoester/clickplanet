import {useCallback, useState} from "react"
import {ClipsAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {StatsSnapshot, TikTokAccount} from "../gen/grpc/clips/v1/admin_pb.ts"
import {saveText} from "../map/download.ts"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {useCall} from "../useCall.ts"
import {Span, Table, tableFileOf, windowOf} from "./actions.ts"

// A fortnight of runs: enough to see a day missed, or the day TikTok's business lane opened.
const SHOWN = 14

const SPANS: {span: Span, label: string}[] = [
    {span: "last", label: "Last snapshot"},
    {span: "week", label: "7 days"},
    {span: "month", label: "30 days"},
    {span: "all", label: "Everything"},
]

export type Save = (name: string, text: string, type: string) => void

// The numbers of the clips the clips job posted, a snapshot a day (`clips stats`), and a table of them for a
// spreadsheet. A read: the job keeps them, over ssh.
export function ClipStatsPage({clips, save = saveText, now = () => new Date()}: {
    clips: ClipsAdmin, save?: Save, now?: () => Date,
}) {
    const list = useCallback(
        async (signal: AbortSignal) => (await clips.listSnapshots({limit: SHOWN}, {signal})).snapshots,
        [clips],
    )
    const [snapshots, again] = useCall(list)

    return (
        <section aria-label="Clip stats">
            <p className="hint">
                Every day at 05:41 UTC, <code>clips stats</code> reads the numbers of every clip of the last 4 weeks from
                Zernio, on each platform, with what the clip was. TikTok sends its business numbers 24 to 48 hours after a
                post, for 7 days.
            </p>
            {snapshots.state === "loading" && <p className="muted">Loading…</p>}
            {snapshots.state === "failed" && <FailureNotice failure={snapshots} onRetry={again}/>}
            {snapshots.state === "done" && (
                <>
                    <Download clips={clips} latest={snapshots.value[0]} save={save} now={now}/>
                    {snapshots.value.length === 0
                        ? <p className="muted">No snapshot yet: the first comes with the next run of the clips stats workflow.</p>
                        : <>
                            <TikTokLanes snapshot={snapshots.value[0]}/>
                            <Snapshots snapshots={snapshots.value}/>
                        </>}
                </>
            )}
        </section>
    )
}

function Download({clips, latest, save, now}: {
    clips: ClipsAdmin, latest: StatsSnapshot | undefined, save: Save, now: () => Date,
}) {
    const [span, setSpan] = useState<Span>("last")
    const [saved, start] = useAction<Table>(wordsOf)

    const download = () => start(async () => {
        const table = await tableFileOf(clips, windowOf(span, latest, now()), now())
        save(table.name, table.csv, "text/csv")
        return table
    })

    return (
        <section className="panel" aria-label="Download the table">
            <p className="hint">
                A row for each platform of each clip at each snapshot: its age, every number Zernio gave, and what the clip
                was (<code>meta.record.clip.variants.*</code> are the arms of the experiments, <code>meta.record.post.variants.*</code>
                the arm a post went out in when the job moved it, and <code>meta.record.post.sound.*</code> the trending track on
                TikTok). The last snapshot is each
                clip once, at its latest age.
            </p>
            <div className="toolbar">
                <div className="segmented" role="group" aria-label="Snapshots">
                    {SPANS.map(choice => (
                        <button
                            key={choice.span}
                            type="button"
                            aria-pressed={span === choice.span}
                            onClick={() => setSpan(choice.span)}
                        >
                            {choice.label}
                        </button>
                    ))}
                </div>
            </div>
            <div className="action">
                <button
                    type="button"
                    className="button"
                    disabled={saved.step === "running" || latest === undefined}
                    onClick={download}
                >
                    {saved.step === "running" ? "Reading…" : "Download CSV"}
                </button>
                {saved.step === "refused" && <p className="action-refused" role="alert">{saved.words}</p>}
                {saved.step === "done" && (
                    <p className="action-said" role="status">Saved {saved.value.name}: {countOf(saved.value.rows, "row")}.</p>
                )}
            </div>
        </section>
    )
}

// Whether TikTok's business lane is live, as the last snapshot saw it.
function TikTokLanes({snapshot}: {snapshot: StatsSnapshot}) {
    if (snapshot.tiktokAccounts.length === 0) return <p className="muted">No TikTok account was connected.</p>
    return (
        <section className="panel" aria-label="TikTok business lane">
            <table className="table">
                <thead>
                    <tr>
                        <th scope="col">TikTok account</th>
                        <th scope="col">Type</th>
                        <th scope="col" className="num">Business numbers</th>
                        <th scope="col">Music library</th>
                        <th scope="col">Business lane</th>
                    </tr>
                </thead>
                <tbody>
                    {snapshot.tiktokAccounts.map(account => <Lane key={account.accountId} account={account}/>)}
                </tbody>
            </table>
        </section>
    )
}

function Lane({account}: {account: TikTokAccount}) {
    return (
        <tr>
            <td>{account.username === "" ? account.accountId : `@${account.username}`}</td>
            <td>{account.accountType === "" ? "not said" : account.accountType}</td>
            <td className="num">{account.businessRows} of {account.rows}</td>
            <td title={account.commercialMusicRefusal}>{account.commercialMusic ? "answers" : "refused"}</td>
            <td>{account.live ? <strong>live</strong> : "not live"}</td>
        </tr>
    )
}

function Snapshots({snapshots}: {snapshots: StatsSnapshot[]}) {
    return (
        <section className="panel" aria-label="Snapshots">
            <table className="table">
                <thead>
                    <tr>
                        <th scope="col">Taken</th>
                        <th scope="col" className="num">Clips</th>
                        <th scope="col" className="num">Rows</th>
                        <th scope="col" className="num">Syncing</th>
                        <th scope="col" className="num">YouTube curves</th>
                        <th scope="col">Lost</th>
                    </tr>
                </thead>
                <tbody>
                    {snapshots.map(snapshot => (
                        <tr key={snapshot.takenAt?.toDate().toISOString()}>
                            <td><When at={snapshot.takenAt}/></td>
                            <td className="num">
                                {snapshot.posts}{snapshot.notOut > 0 && <span className="muted"> ({snapshot.notOut} not out)</span>}
                            </td>
                            <td className="num">{snapshot.rows}</td>
                            <td className="num">{snapshot.syncing}</td>
                            <td className="num">{snapshot.curves}</td>
                            <td title={snapshot.unread.map(post => `${post.postId}: ${post.reason}`).join("\n")}>
                                {snapshot.unread.length === 0 ? "" : countOf(snapshot.unread.length, "clip")}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </section>
    )
}
