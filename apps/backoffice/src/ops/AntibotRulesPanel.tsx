import {useCallback, useState} from "react"
import {AntibotAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse, RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {bansOf, crowdOf, Draft, draftOf, flawOf, newDraft, offOf, saveRuleSet} from "./antibotRules.ts"

type Editing = {draft: Draft, made: boolean}

// Every antibot rule set postgres keeps, and a JSON editor to make one or change one.
export function AntibotRulesPanel({antibot}: {antibot: AntibotAdmin}) {
    const read = useCallback((signal: AbortSignal) => antibot.getRules({}, {signal}), [antibot])
    const [rules, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const saved = () => {
        setEditing(undefined)
        again()
    }

    return (
        <section aria-label="Anti-bot rules">
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && (
                <>
                    <p className="hint">Playing: <code>{rules.value.playing}</code></p>
                    <Sets rules={rules.value} onEdit={set => setEditing({draft: draftOf(set), made: true})}/>
                    <button
                        type="button"
                        className="button"
                        onClick={() => setEditing({draft: newDraft(rules.value), made: false})}
                    >
                        New set
                    </button>
                </>
            )}
            {editing !== undefined && (
                <SetForm
                    key={editing.made ? editing.draft.name : "new"}
                    antibot={antibot}
                    editing={editing}
                    onCancel={() => setEditing(undefined)}
                    onSaved={saved}
                />
            )}
        </section>
    )
}

function Sets({rules, onEdit}: {rules: GetRulesResponse, onEdit: (set: RuleSet) => void}) {
    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr><th>Set</th><th>Bans</th><th>Watchdogs off</th><th>Crowd</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{bansOf(set)}</td>
                        <td>{offOf(set)}</td>
                        <td>{crowdOf(set)}</td>
                        <td>
                            <button type="button" className="button-mini" onClick={() => onEdit(set)}>
                                Edit {set.name}
                            </button>
                        </td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

function SetForm({antibot, editing, onCancel, onSaved}: {
    antibot: AntibotAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const flaw = flawOf(draft.text)
    const title = editing.made ? `Edit ${editing.draft.name}` : "New set"

    return (
        <section className="panel" aria-label={title}>
            <h2>{title}</h2>
            <label className="field">
                <span className="muted">Name</span>
                <input
                    value={draft.name}
                    disabled={editing.made}
                    onChange={e => setDraft(was => ({...was, name: e.target.value}))}
                />
            </label>
            <label className="field">
                <span className="muted">Rules (JSON)</span>
                <textarea
                    className="mono rules-json"
                    spellCheck={false}
                    value={draft.text}
                    onChange={e => setDraft(was => ({...was, text: e.target.value}))}
                />
            </label>
            {flaw !== undefined && <p className="action-refused" role="alert">{flaw}</p>}
            <ConfirmAction
                label="Save"
                disabled={draft.name.trim() === "" || flaw !== undefined}
                question={`Save ${draft.name.trim()}?`}
                run={() => saveRuleSet(antibot, draft)}
                said={() => "Saved."}
                refusal={wordsOf}
                onDone={onSaved}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
