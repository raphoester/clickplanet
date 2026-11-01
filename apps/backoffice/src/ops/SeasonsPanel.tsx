import {ConnectError, Code} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {AntibotAdmin, GamemasterAdmin, PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {zoneOf} from "../components/time.ts"
import {When} from "../components/When.tsx"
import {GetSeasonsResponse} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {localized, wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {Cutoff, cutoffOf, daysEndWords} from "./cutoff.ts"
import {Modules, playable} from "./schedule.ts"
import {
    addSeason, complete, deletable, deleteSeason, Draft, draftOf, editSeason, FinaleDraft, finaleStarted, finaleStartOf,
    finaleSummaryOf, finaleWindowOf, flawsOf, phaseOf, questionOf, Row, rowsOf, spanOf, spansOf, windowOf,
} from "./seasons.ts"

type Read = Modules & {seasons: GetSeasonsResponse}

// The server says why: a season that started, or a window that ends before now.
const refusal = (err: unknown) => {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `The server refused it: ${localized(error.rawMessage)}`
    return wordsOf(err, {notFound: "This season is no longer kept."})
}

// Editing: a season that has not ended, or undefined for a new one.
type Editing = {row: Row | undefined}

export function SeasonsPanel({gamemaster, planet, antibot}: {
    gamemaster: GamemasterAdmin, planet: PlanetAdmin, antibot: AntibotAdmin,
}) {
    const read = useCallback(async (signal: AbortSignal): Promise<Read> => {
        const [seasons, planetRules, antibotRules] = await Promise.all([
            gamemaster.getSeasons({}, {signal}), planet.getRules({}, {signal}), antibot.getRules({}, {signal}),
        ])
        return {seasons, planet: planetRules, antibot: antibotRules}
    }, [gamemaster, planet, antibot])
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
                        rows={rowsOf(state.value.seasons)}
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
                            rows={rowsOf(state.value.seasons)}
                            sets={playable(state.value)}
                            cutoff={cutoffOf(state.value.seasons.dailyCutoff)}
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

function SeasonForm({gamemaster, row, rows, sets, cutoff, onCancel, onDone}: {
    gamemaster: GamemasterAdmin, row: Row | undefined, rows: Row[], sets: string[], cutoff: Cutoff | undefined,
    onCancel: () => void, onDone: () => void,
}) {
    const [draft, setDraft] = useState(() => draftOf(row, rows))
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const now = new Date()
    const title = row === undefined ? "New season" : `Edit season ${row.number}`
    const locked = row !== undefined && phaseOf(row, now) === "running"
    const ending = row !== undefined && finaleStarted(row, now)
    const window = windowOf(draft, row, rows)
    const spans = spansOf(draft, row, rows)

    return (
        <section className="panel" aria-label={title}>
            <h2>{title}</h2>
            <p className="hint">Times are in {zoneOf()}.</p>
            <div className="fields">
                <label className="field">
                    <span className="muted">Starts</span>
                    <input type="datetime-local" value={draft.startsAt} disabled={locked} onChange={e => change({startsAt: e.target.value})}/>
                </label>
                <label className="field">
                    <span className="muted">Ends</span>
                    <input type="datetime-local" value={draft.endsAt} disabled={ending} onChange={e => change({endsAt: e.target.value})}/>
                </label>
            </div>
            {spans !== undefined && <p className="hint">{spans}.</p>}
            {cutoff !== undefined && <p className="hint">{daysEndWords(window?.endsAt ?? now, cutoff)}</p>}
            <FinaleFields
                finale={draft.finale}
                sets={sets}
                window={finaleWindowOf(draft, row, rows)}
                disabled={ending}
                onChange={finale => change({finale})}
            />
            {flawsOf(draft, row, rows, cutoff, now).map(flaw => <p key={flaw} className="action-refused" role="alert">{flaw}</p>)}
            <ConfirmAction
                label={row === undefined ? "Add" : "Save"}
                disabled={!complete(draft, row, rows)}
                question={questionOf(draft, row, rows)}
                run={async (): Promise<number | undefined> => {
                    if (row === undefined) return addSeason(gamemaster, draft, rows)
                    await editSeason(gamemaster, row, draft, rows)
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

function FinaleFields({finale, sets, window, disabled, onChange}: {
    finale: FinaleDraft, sets: string[], window: {startsAt: Date, endsAt: Date} | undefined, disabled: boolean,
    onChange: (finale: FinaleDraft) => void,
}) {
    const change = (part: Partial<FinaleDraft>) => onChange({...finale, ...part})
    const options = sets.includes(finale.ruleSet) || finale.ruleSet === "" ? sets : [finale.ruleSet, ...sets]
    return (
        <>
            <fieldset className="fields" aria-label="Final battle">
                <label className="field">
                    <span className="muted">Ends with a final battle</span>
                    <input type="checkbox" checked={finale.on} disabled={disabled} onChange={e => change({on: e.target.checked})}/>
                </label>
                {finale.on && (
                    <>
                        <label className="field">
                            <span className="muted">Final battle, hours</span>
                            <input type="number" inputMode="decimal" min={0} value={finale.hours} disabled={disabled}
                                onChange={e => change({hours: e.target.value})}/>
                        </label>
                        <label className="field">
                            <span className="muted">Rule set</span>
                            <select value={finale.ruleSet} disabled={disabled} onChange={e => change({ruleSet: e.target.value})}>
                                {finale.ruleSet === "" && <option value="">Pick a set</option>}
                                {options.map(set => <option key={set} value={set}>{set}</option>)}
                            </select>
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
            {finale.on && window !== undefined && (
                <p className="hint">Final battle from <When at={window.startsAt}/> to <When at={window.endsAt}/></p>
            )}
        </>
    )
}
