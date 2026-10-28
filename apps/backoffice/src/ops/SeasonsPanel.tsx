import {ConnectError, Code} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {GamemasterAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {
    addSeason, complete, deletable, deleteSeason, Draft, draftOf, editSeason, FinaleDraft, finaleStarted, finaleStartOf,
    finaleSummaryOf, phaseOf, questionOf, Row, rowsOf, Span, spanOf, windowOf,
} from "./seasons.ts"

// The server says why: a season that started, or a window that ends before now.
const refusal = (err: unknown) => {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `The server refused it: ${error.rawMessage}`
    return wordsOf(err, {notFound: "This season is no longer kept."})
}

// Editing: a season that has not ended, or undefined for a new one.
type Editing = {row: Row | undefined}

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
                        onEdit={row => setEditing({row})}
                        onDeleted={changed}
                    />
                    {editing === undefined && (
                        <button type="button" className="button" onClick={() => setEditing({row: undefined})}>
                            Add a season
                        </button>
                    )}
                    {editing !== undefined && (
                        <SeasonForm
                            key={editing.row?.number ?? "new"}
                            gamemaster={gamemaster}
                            row={editing.row}
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
    rows: Row[], gamemaster: GamemasterAdmin, onEdit: (row: Row) => void, onDeleted: () => void,
}) {
    const now = new Date()

    return (
        <table className="table">
            <caption>Seasons</caption>
            <thead>
                <tr><th>Season</th><th>Break</th><th>From</th><th>Length</th><th>To</th><th>Final battle</th><th>State</th><th/></tr>
            </thead>
            <tbody>
                {rows.map(row => {
                    const phase = phaseOf(row, now)
                    return (
                        <tr
                            key={row.number}
                            aria-current={phase === "running" ? "true" : undefined}
                            className={phase === "done" ? "muted" : undefined}
                        >
                            <td><strong>{row.number}</strong></td>
                            <td>{spanOf(row.scheduled.breakBefore)}</td>
                            <td><When at={row.startsAt}/></td>
                            <td>{spanOf(row.scheduled.length)}</td>
                            <td><When at={row.endsAt}/></td>
                            <td>
                                {finaleSummaryOf(row.scheduled.finale)}
                                {finaleStartOf(row) !== undefined && <> from <When at={finaleStartOf(row)}/></>}
                            </td>
                            <td>{phase}</td>
                            <td>
                                {phase !== "done" && (
                                    <button type="button" className="button-mini" onClick={() => onEdit(row)}>Edit</button>
                                )}
                                {deletable(row, rows, now) && (
                                    <ConfirmAction
                                        label="Delete"
                                        tone="danger"
                                        question={`Delete season ${row.number}, from ${row.startsAt?.toLocaleString()}?`}
                                        run={() => deleteSeason(gamemaster, row.number)}
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

function SeasonForm({gamemaster, row, rows, onCancel, onDone}: {
    gamemaster: GamemasterAdmin, row: Row | undefined, rows: Row[], onCancel: () => void, onDone: () => void,
}) {
    const [draft, setDraft] = useState(() => draftOf(row?.scheduled))
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const title = row === undefined ? "New season" : `Edit season ${row.number}`
    const locked = row !== undefined && phaseOf(row, new Date()) === "running"
    const ending = row !== undefined && finaleStarted(row, new Date())
    const window = windowOf(draft, row, rows)

    return (
        <section className="panel" aria-label={title}>
            <h2>{title}</h2>
            <div className="fields">
                <SpanFields label="Break" span={draft.breakBefore} disabled={locked} onChange={breakBefore => change({breakBefore})}/>
                <SpanFields label="Length" span={draft.length} disabled={ending} onChange={length => change({length})}/>
            </div>
            <FinaleFields finale={draft.finale} disabled={ending} onChange={finale => change({finale})}/>
            {window !== undefined && (
                <p className="hint">From <When at={window.startsAt}/> to <When at={window.endsAt}/></p>
            )}
            <ConfirmAction
                label={row === undefined ? "Add" : "Save"}
                disabled={!complete(draft)}
                question={questionOf(draft, row, rows)}
                run={async (): Promise<number | undefined> => {
                    if (row === undefined) return addSeason(gamemaster, draft)
                    await editSeason(gamemaster, row, draft)
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

function FinaleFields({finale, disabled, onChange}: {
    finale: FinaleDraft, disabled: boolean, onChange: (finale: FinaleDraft) => void,
}) {
    const change = (part: Partial<FinaleDraft>) => onChange({...finale, ...part})
    return (
        <fieldset className="fields" aria-label="Final battle">
            <label className="field">
                <span className="muted">Ends with a final battle</span>
                <input type="checkbox" checked={finale.on} disabled={disabled} onChange={e => change({on: e.target.checked})}/>
            </label>
            {finale.on && (
                <>
                    <SpanFields label="Final battle" span={finale.length} disabled={disabled} onChange={length => change({length})}/>
                    <label className="field">
                        <span className="muted">Rule set</span>
                        <input value={finale.ruleSet} disabled={disabled} onChange={e => change({ruleSet: e.target.value})}/>
                    </label>
                    <label className="field">
                        <span className="muted">Times</span>
                        <input type="number" inputMode="numeric" min={1} max={10} step={1} value={finale.times} disabled={disabled}
                            onChange={e => change({times: e.target.value})}/>
                    </label>
                    <label className="field">
                        <span className="muted">Description</span>
                        <textarea maxLength={300} value={finale.description} disabled={disabled}
                            onChange={e => change({description: e.target.value})}/>
                    </label>
                </>
            )}
        </fieldset>
    )
}

function SpanFields({label, span, disabled, onChange}: {
    label: string, span: Span, disabled: boolean, onChange: (span: Span) => void,
}) {
    return (
        <>
            <label className="field">
                <span className="muted">{label}, days</span>
                <input type="number" inputMode="numeric" min={0} step={1} value={span.days} disabled={disabled}
                    onChange={e => onChange({...span, days: e.target.value})}/>
            </label>
            <label className="field">
                <span className="muted">{label}, hours</span>
                <input type="number" inputMode="decimal" min={0} value={span.hours} disabled={disabled}
                    onChange={e => onChange({...span, hours: e.target.value})}/>
            </label>
        </>
    )
}
