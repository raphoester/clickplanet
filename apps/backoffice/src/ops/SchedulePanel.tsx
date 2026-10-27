import {ConnectError, Code} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {AntibotAdmin, PlanetAdmin, GamemasterAdmin, ScoringAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {Entry, GetScheduleResponse} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {EntryKind} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {
    addEntry, changeable, complete, deleteEntry, Draft, draftOf, editEntry, ended, kinds, lacking, MAX_DESCRIPTION, Modules, playable,
    questionOf, Row, rowsOf, shortOf, wordOf,
} from "./schedule.ts"

type Read = Modules & {schedule: GetScheduleResponse}

// The server says which module lacks the set, or since when the entry runs.
const refusal = (err: unknown) => {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `The server refused it: ${error.rawMessage}`
    return wordsOf(err, {notFound: "This entry is no longer kept."})
}

// Editing: an entry to come, or undefined for a new one.
type Editing = {entry: Entry | undefined}

export function SchedulePanel({gamemaster, planet, antibot, scoring}: {
    gamemaster: GamemasterAdmin, planet: PlanetAdmin, antibot: AntibotAdmin, scoring: ScoringAdmin,
}) {
    const read = useCallback(async (signal: AbortSignal): Promise<Read> => {
        const [schedule, planetRules, antibotRules, seasonRules] = await Promise.all([
            gamemaster.getSchedule({}, {signal}), planet.getRules({}, {signal}), antibot.getRules({}, {signal}),
            scoring.getRules({}, {signal}),
        ])
        return {schedule, planet: planetRules, antibot: antibotRules, scoring: seasonRules}
    }, [gamemaster, planet, antibot, scoring])
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
                        gamemaster={gamemaster}
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
                            gamemaster={gamemaster}
                            entry={editing.entry}
                            sets={playable(state.value)}
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
                Playing: <code>{set}</code>, a {wordOf(playing?.kind ?? EntryKind.PLAIN)} entry
                {playing?.since !== undefined && <>, since <When at={playing.since}/></>}
            </p>
            <p className="hint">
                Planet plays <code>{read.planet.playing}</code>, the antibot plays <code>{read.antibot.playing}</code>,
                scoring plays <code>{read.scoring.playing}</code>.
            </p>
            {read.planet.playing !== set && (
                <p className="action-refused" role="alert">Planet does not play <code>{set}</code>.</p>
            )}
            {read.antibot.playing !== set && (
                <p className="action-refused" role="alert">The antibot does not play <code>{set}</code>.</p>
            )}
            {read.scoring.playing !== set && (
                <p className="action-refused" role="alert">Scoring does not play <code>{set}</code>.</p>
            )}
        </>
    )
}

function Rows({read, gamemaster, onEdit, onDeleted}: {
    read: Read, gamemaster: GamemasterAdmin, onEdit: (entry: Entry) => void, onDeleted: () => void,
}) {
    const now = new Date()
    const playing = read.schedule.playing?.entryId ?? ""

    return (
        <table className="table">
            <caption>Schedule</caption>
            <thead>
                <tr><th>From</th><th>To</th><th>Set</th><th>Kind</th><th>Description</th><th>Priority</th><th/></tr>
            </thead>
            <tbody>
                {rowsOf(read.schedule).map(row => (
                    <tr
                        key={row.entry.id}
                        aria-current={row.entry.id === playing ? "true" : undefined}
                        className={ended(row, now) ? "muted" : undefined}
                    >
                        <td><When at={row.startsAt}/></td>
                        <td><When at={row.endsAt}/></td>
                        <td>
                            <strong>{row.entry.ruleSet}</strong>
                            {!ended(row, now) && <Lacking set={row.entry.ruleSet} read={read}/>}
                        </td>
                        <td>{wordOf(row.entry.kind)}</td>
                        <td title={row.entry.description}>{shortOf(row.entry.description)}</td>
                        <td>{row.entry.priority}</td>
                        <td>{changeable(row, now) && (
                            <EntryActions row={row} gamemaster={gamemaster} onEdit={onEdit} onDeleted={onDeleted}/>
                        )}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

function Lacking({set, read}: {set: string, read: Read}) {
    const missing = lacking(set, read)
    if (missing.length === 0) return null
    const named = missing.length === 1 ? missing[0] : `${missing.slice(0, -1).join(", ")} or ${missing[missing.length - 1]}`
    return <p className="action-refused" role="alert">No set {set} in {named}.</p>
}

function EntryActions({row, gamemaster, onEdit, onDeleted}: {
    row: Row, gamemaster: GamemasterAdmin, onEdit: (entry: Entry) => void, onDeleted: () => void,
}) {
    return (
        <>
            <button type="button" className="button-mini" onClick={() => onEdit(row.entry)}>Edit</button>
            <ConfirmAction
                label="Delete"
                tone="danger"
                question={`Delete the entry ${row.entry.ruleSet}, from ${row.startsAt?.toLocaleString()}?`}
                run={() => deleteEntry(gamemaster, row.entry.id)}
                said={() => "Deleted."}
                refusal={refusal}
                onDone={onDeleted}
            />
        </>
    )
}

function EntryForm({gamemaster, entry, sets, onCancel, onDone}: {
    gamemaster: GamemasterAdmin, entry: Entry | undefined, sets: string[], onCancel: () => void, onDone: () => void,
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
                    <span className="muted">Kind</span>
                    <select value={String(draft.kind)} onChange={e => change({kind: Number(e.target.value) as EntryKind})}>
                        {kinds().map(kind => <option key={kind} value={String(kind)}>{wordOf(kind)}</option>)}
                    </select>
                </label>
                <label className="field">
                    <span className="muted">Description</span>
                    <textarea rows={2} maxLength={MAX_DESCRIPTION} value={draft.description}
                        onChange={e => change({description: e.target.value})}/>
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
                    <input type="number" inputMode="numeric" min={0} value={draft.priority}
                        onChange={e => change({priority: e.target.value})}/>
                </label>
            </div>
            <ConfirmAction
                label={entry === undefined ? "Add" : "Save"}
                disabled={!complete(draft)}
                question={questionOf(draft)}
                run={() => entry === undefined ? addEntry(gamemaster, draft) : editEntry(gamemaster, entry.id, draft)}
                said={() => "Scheduled."}
                refusal={refusal}
                onDone={onDone}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
