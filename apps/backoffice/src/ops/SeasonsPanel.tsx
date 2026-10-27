import {ConnectError, Code} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {GamemasterAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {Season} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {
    addSeason, complete, deletable, deleteSeason, Draft, draftOf, editSeason, phaseOf, questionOf, Row, rowsOf, started,
} from "./seasons.ts"

// The server says why: a season that started, or a window that overlaps another.
const refusal = (err: unknown) => {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `The server refused it: ${error.rawMessage}`
    return wordsOf(err, {notFound: "This season is no longer kept."})
}

// Editing: a season that has not ended, or undefined for a new one.
type Editing = {season: Season | undefined}

export function SeasonsPanel({gamemaster}: {gamemaster: GamemasterAdmin}) {
    const read = useCallback((signal: AbortSignal) => gamemaster.getSeasons({}, {signal}), [gamemaster])
    const [state, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const changed = () => {
        setEditing(undefined)
        again()
    }

    return (
        <section aria-label="Seasons">
            {state.state === "loading" && <p className="muted">Loading…</p>}
            {state.state === "failed" && <FailureNotice failure={state} onRetry={again}/>}
            {state.state === "done" && (
                <>
                    <Rows
                        rows={rowsOf(state.value)}
                        gamemaster={gamemaster}
                        onEdit={season => setEditing({season})}
                        onDeleted={changed}
                    />
                    {editing === undefined && (
                        <button type="button" className="button" onClick={() => setEditing({season: undefined})}>
                            Add a season
                        </button>
                    )}
                    {editing !== undefined && (
                        <SeasonForm
                            key={editing.season?.number ?? "new"}
                            gamemaster={gamemaster}
                            season={editing.season}
                            rows={rowsOf(state.value)}
                            onCancel={() => setEditing(undefined)}
                            onDone={changed}
                        />
                    )}
                </>
            )}
        </section>
    )
}

function Rows({rows, gamemaster, onEdit, onDeleted}: {
    rows: Row[], gamemaster: GamemasterAdmin, onEdit: (season: Season) => void, onDeleted: () => void,
}) {
    const now = new Date()

    return (
        <table className="table">
            <caption>Seasons</caption>
            <thead>
                <tr><th>Season</th><th>From</th><th>To</th><th>State</th><th/></tr>
            </thead>
            <tbody>
                {rows.map(row => {
                    const phase = phaseOf(row, now)
                    return (
                        <tr
                            key={row.season.number}
                            aria-current={phase === "running" ? "true" : undefined}
                            className={phase === "done" ? "muted" : undefined}
                        >
                            <td><strong>{row.season.number}</strong></td>
                            <td><When at={row.startsAt}/></td>
                            <td><When at={row.endsAt}/></td>
                            <td>{phase}</td>
                            <td>
                                {phase !== "done" && (
                                    <button type="button" className="button-mini" onClick={() => onEdit(row.season)}>Edit</button>
                                )}
                                {deletable(row, rows, now) && (
                                    <ConfirmAction
                                        label="Delete"
                                        tone="danger"
                                        question={`Delete season ${row.season.number}, from ${row.startsAt?.toLocaleString()}?`}
                                        run={() => deleteSeason(gamemaster, row.season.number)}
                                        said={() => "Deleted."}
                                        refusal={refusal}
                                        onDone={onDeleted}
                                    />
                                )}
                            </td>
                        </tr>
                    )
                })}
            </tbody>
        </table>
    )
}

function SeasonForm({gamemaster, season, rows, onCancel, onDone}: {
    gamemaster: GamemasterAdmin, season: Season | undefined, rows: Row[], onCancel: () => void, onDone: () => void,
}) {
    const [draft, setDraft] = useState(() => draftOf(season, rows))
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const title = season === undefined ? "New season" : `Edit season ${season.number}`
    const locked = season !== undefined && started({season, startsAt: season.startsAt?.toDate()}, new Date())

    return (
        <section className="panel" aria-label={title}>
            <h2>{title}</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Starts</span>
                    <input type="datetime-local" value={draft.startsAt} disabled={locked}
                        onChange={e => change({startsAt: e.target.value})}/>
                </label>
                <label className="field">
                    <span className="muted">Ends</span>
                    <input type="datetime-local" value={draft.endsAt} onChange={e => change({endsAt: e.target.value})}/>
                </label>
            </div>
            <ConfirmAction
                label={season === undefined ? "Add" : "Save"}
                disabled={!complete(draft)}
                question={questionOf(draft, season)}
                run={async (): Promise<number | undefined> => {
                    if (season === undefined) return addSeason(gamemaster, draft)
                    await editSeason(gamemaster, season, draft)
                    return undefined
                }}
                said={number => number === undefined ? "Saved." : `Season ${number} added.`}
                refusal={refusal}
                onDone={onDone}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
