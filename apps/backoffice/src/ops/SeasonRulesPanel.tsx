import {useCallback, useState} from "react"
import {ScoringAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse, RuleSet} from "../gen/grpc/scoring/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {complete, copyOf, Draft, draftOf, MAX_TIMES, roundOf, saveRuleSet} from "./seasonRules.ts"

// made: an edit of a set kept, with its name fixed; otherwise a new set copied from `from`.
type Editing = {draft: Draft, made: boolean, from: string}

const refusal = (err: unknown) =>
    wordsOf(err, {failedPrecondition: "A set of this name plays now. Save it under another name."})

// Every season rule set postgres keeps, and a form to make one or change one.
export function SeasonRulesPanel({scoring}: {scoring: ScoringAdmin}) {
    const read = useCallback((signal: AbortSignal) => scoring.getRules({}, {signal}), [scoring])
    const [rules, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const saved = () => {
        setEditing(undefined)
        again()
    }

    return (
        <section aria-label="Season rules">
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && (
                <>
                    <p className="hint">Playing: <code>{rules.value.playing}</code></p>
                    <p className="hint">
                        A round of its own is the window of the schedule entry that plays the set, scored from the map
                        at its end. The one that ends with the season is its Final Battle.
                    </p>
                    <Sets
                        rules={rules.value}
                        onEdit={set => setEditing({draft: draftOf(set), made: true, from: set.name})}
                        onCopy={set => setEditing({draft: copyOf(set), made: false, from: set.name})}
                    />
                </>
            )}
            {editing !== undefined && (
                <SetForm
                    key={`${editing.made ? "edit" : "copy"} ${editing.from}`}
                    scoring={scoring}
                    editing={editing}
                    onCancel={() => setEditing(undefined)}
                    onSaved={saved}
                />
            )}
        </section>
    )
}

function Sets({rules, onEdit, onCopy}: {
    rules: GetRulesResponse, onEdit: (set: RuleSet) => void, onCopy: (set: RuleSet) => void,
}) {
    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr><th>Set</th><th>Round</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{roundOf(set)}</td>
                        <td>
                            {set.name !== rules.playing && (
                                <button type="button" className="button-mini" onClick={() => onEdit(set)}>
                                    Edit {set.name}
                                </button>
                            )}
                            <button type="button" className="button-mini" onClick={() => onCopy(set)}>
                                Copy {set.name}
                            </button>
                        </td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

function SetForm({scoring, editing, onCancel, onSaved}: {
    scoring: ScoringAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const title = editing.made ? `Edit ${editing.from}` : `New set from ${editing.from}`

    return (
        <section className="panel" aria-label={title}>
            <h2>{title}</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Name</span>
                    <input
                        value={draft.name}
                        disabled={editing.made}
                        onChange={e => setDraft(was => ({...was, name: e.target.value}))}
                    />
                </label>
                <label className="field">
                    <span className="muted">A round of its own</span>
                    <input
                        type="checkbox"
                        checked={draft.ownRound}
                        onChange={e => setDraft(was => ({...was, ownRound: e.target.checked}))}
                    />
                </label>
                <label className="field">
                    <span className="muted">Points ×</span>
                    <input
                        type="number"
                        inputMode="numeric"
                        min={1}
                        max={MAX_TIMES}
                        value={draft.times}
                        onChange={e => setDraft(was => ({...was, times: e.target.value}))}
                    />
                </label>
            </div>
            <ConfirmAction
                label="Save"
                disabled={!complete(draft)}
                question={`Save ${draft.name.trim()}?`}
                run={() => saveRuleSet(scoring, draft)}
                said={() => "Saved."}
                refusal={refusal}
                onDone={onSaved}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
