import {ConnectError, Code} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {AntibotAdmin, PlanetAdmin, SeasonsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {GetRulesResponse as AntibotRules} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {GetRulesResponse as PlanetRules} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Entry, GetScheduleResponse} from "../gen/grpc/seasons/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {
    addEntry, changeable, complete, deleteEntry, Draft, draftOf, editEntry, ended, lacking, phaseName, playable, playingOf,
    questionOf, Row, rowsOf,
} from "./schedule.ts"

type Read = {schedule: GetScheduleResponse, planet: PlanetRules, antibot: AntibotRules}

// The server says which module lacks the set, or since when the entry runs.
const refusal = (err: unknown) => {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `The server refused it: ${error.rawMessage}`
    return wordsOf(err, {notFound: "This entry is no longer kept."})
}

// Editing: an entry to come, or undefined for a new one.
type Editing = {entry: Entry | undefined}

export function SchedulePanel({seasons, planet, antibot}: {seasons: SeasonsAdmin, planet: PlanetAdmin, antibot: AntibotAdmin}) {
    const read = useCallback(async (signal: AbortSignal): Promise<Read> => {
        const [schedule, planetRules, antibotRules] = await Promise.all([
            seasons.getSchedule({}, {signal}), planet.getRules({}, {signal}), antibot.getRules({}, {signal}),
        ])
        return {schedule, planet: planetRules, antibot: antibotRules}
    }, [seasons, planet, antibot])
    const [state, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const changed = () => {
        setEditing(undefined)
        again()
    }

    return (
        <section aria-label="Rule schedule">
            {state.state === "loading" && <p className="muted">Loading…</p>}
            {state.state === "failed" && <FailureNotice failure={state} onRetry={again}/>}
            {state.state === "done" && (
                <>
                    <Playing read={state.value}/>
                    <Rows
                        read={state.value}
                        seasons={seasons}
                        onEdit={entry => setEditing({entry})}
                        onDeleted={changed}
                    />
                    {editing === undefined && (
                        <button type="button" className="button" onClick={() => setEditing({entry: undefined})}>
                            Add an entry
                        </button>
                    )}
                    {editing !== undefined && (
                        <EntryForm
                            key={editing.entry?.id ?? "new"}
                            seasons={seasons}
                            entry={editing.entry}
                            sets={playable(state.value.planet, state.value.antibot)}
                            onCancel={() => setEditing(undefined)}
                            onDone={changed}
                        />
                    )}
                </>
            )}
        </section>
    )
}

function Playing({read}: {read: Read}) {
    const playing = read.schedule.playing
    const set = playing?.ruleSet ?? "default"
    return (
        <>
            <p className="hint">
                Playing: <code>{set}</code>{playing?.since !== undefined && <> since <When at={playing.since}/></>}
            </p>
            <p className="hint">
                Planet plays <code>{read.planet.playing}</code>, the antibot plays <code>{read.antibot.playing}</code>.
            </p>
            {read.planet.playing !== set && (
                <p className="action-refused" role="alert">Planet does not play <code>{set}</code>.</p>
            )}
            {read.antibot.playing !== set && (
                <p className="action-refused" role="alert">The antibot does not play <code>{set}</code>.</p>
            )}
        </>
    )
}

function Rows({read, seasons, onEdit, onDeleted}: {
    read: Read, seasons: SeasonsAdmin, onEdit: (entry: Entry) => void, onDeleted: () => void,
}) {
    const now = new Date()
    const rows = rowsOf(read.schedule)
    const playing = playingOf(read.schedule, rows, now)

    return (
        <table className="table">
            <caption>Schedule</caption>
            <thead>
                <tr><th>From</th><th>To</th><th>Set</th><th>Priority</th><th>What</th><th/></tr>
            </thead>
            <tbody>
                {rows.map(row => (
                    <tr
                        key={row.key}
                        aria-current={row.key === playing ? "true" : undefined}
                        className={ended(row, now) ? "muted" : undefined}
                    >
                        <td>{row.startsAt === undefined ? "—" : <When at={row.startsAt}/>}</td>
                        <td>{row.endsAt === undefined ? "—" : <When at={row.endsAt}/>}</td>
                        <td>
                            <strong>{row.set}</strong>
                            {!ended(row, now) && <Lacking row={row} read={read}/>}
                        </td>
                        <td>{row.priority}</td>
                        <td>{row.phase !== undefined ? phaseName(row.phase) : "Entry"}</td>
                        <td>{row.entry !== undefined && changeable(row, now) && (
                            <EntryActions row={row} entry={row.entry} seasons={seasons} onEdit={onEdit} onDeleted={onDeleted}/>
                        )}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

function Lacking({row, read}: {row: Row, read: Read}) {
    const missing = lacking(row.set, read.planet, read.antibot)
    if (missing.length === 0) return null
    return <p className="action-refused" role="alert">No set {row.set} in {missing.join(" or ")}.</p>
}

function EntryActions({row, entry, seasons, onEdit, onDeleted}: {
    row: Row, entry: Entry, seasons: SeasonsAdmin, onEdit: (entry: Entry) => void, onDeleted: () => void,
}) {
    return (
        <>
            <button type="button" className="button-mini" onClick={() => onEdit(entry)}>Edit</button>
            <ConfirmAction
                label="Delete"
                tone="danger"
                question={`Delete the entry ${row.set}, from ${row.startsAt?.toLocaleString()}?`}
                run={() => deleteEntry(seasons, entry.id)}
                said={() => "Deleted."}
                refusal={refusal}
                onDone={onDeleted}
            />
        </>
    )
}

function EntryForm({seasons, entry, sets, onCancel, onDone}: {
    seasons: SeasonsAdmin, entry: Entry | undefined, sets: string[], onCancel: () => void, onDone: () => void,
}) {
    const [draft, setDraft] = useState(() => draftOf(entry, sets))
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const title = entry === undefined ? "New entry" : `Edit the entry ${entry.ruleSet}`
    const options = sets.includes(draft.set) || draft.set === "" ? sets : [draft.set, ...sets]

    return (
        <section className="panel" aria-label={title}>
            <h2>{title}</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Set</span>
                    <select value={draft.set} onChange={e => change({set: e.target.value})}>
                        {options.map(set => <option key={set} value={set}>{set}</option>)}
                    </select>
                </label>
                <label className="field">
                    <span className="muted">Starts</span>
                    <input type="datetime-local" value={draft.startsAt} onChange={e => change({startsAt: e.target.value})}/>
                </label>
                <label className="field">
                    <span className="muted">Ends</span>
                    <input type="datetime-local" value={draft.endsAt} onChange={e => change({endsAt: e.target.value})}/>
                </label>
                <label className="field">
                    <span className="muted">Priority</span>
                    <input type="number" inputMode="numeric" min={1} value={draft.priority}
                        onChange={e => change({priority: e.target.value})}/>
                </label>
            </div>
            <ConfirmAction
                label={entry === undefined ? "Add" : "Save"}
                disabled={!complete(draft)}
                question={questionOf(draft)}
                run={() => entry === undefined ? addEntry(seasons, draft) : editEntry(seasons, entry.id, draft)}
                said={() => "Scheduled."}
                refusal={refusal}
                onDone={onDone}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
