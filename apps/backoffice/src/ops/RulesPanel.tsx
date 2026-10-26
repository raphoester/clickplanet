import {useCallback, useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {boxesOf, complete, DEFAULT_SET, Draft, draftOf, NEW_DRAFT, saveRuleSet, tempoOf} from "./rules.ts"

type Editing = {draft: Draft, made: boolean}

// Every rule set postgres keeps, and a form to make one or change one.
export function RulesPanel({planet}: {planet: PlanetAdmin}) {
    const read = useCallback((signal: AbortSignal) => planet.getRules({}, {signal}), [planet])
    const [rules, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const saved = () => {
        setEditing(undefined)
        again()
    }

    return (
        <section aria-label="Game rules">
            <p className="hint">
                The game plays <code>{DEFAULT_SET}</code>, and follows a change of it within a second, for every player.
            </p>
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && (
                <Sets rules={rules.value} onEdit={set => setEditing({draft: draftOf(set), made: true})}/>
            )}
            <button type="button" className="button" onClick={() => setEditing({draft: NEW_DRAFT, made: false})}>
                New set
            </button>
            {editing !== undefined && (
                <SetForm
                    key={editing.made ? editing.draft.name : "new"}
                    planet={planet}
                    editing={editing}
                    onCancel={() => setEditing(undefined)}
                    onSaved={saved}
                />
            )}
        </section>
    )
}

function Sets({rules, onEdit}: {rules: GetRulesResponse, onEdit: (set: GetRulesResponse["sets"][number]) => void}) {
    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr><th>Set</th><th>Refill</th><th>Boxes</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === DEFAULT_SET ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{tempoOf(set)}</td>
                        <td>{boxesOf(set.boxes)}</td>
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

const FIELDS: {key: Exclude<keyof Draft, "name">, label: string}[] = [
    {key: "tempo", label: "Refill, times as fast"},
    {key: "minSeconds", label: "Shortest box wait (s)"},
    {key: "maxSeconds", label: "Longest box wait (s)"},
    {key: "cap", label: "Charges an hour, at most"},
]

function SetForm({planet, editing, onCancel, onSaved}: {
    planet: PlanetAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))

    return (
        <section className="panel" aria-label={editing.made ? `Edit ${editing.draft.name}` : "New set"}>
            <h2>{editing.made ? `Edit ${editing.draft.name}` : "New set"}</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Name</span>
                    <input value={draft.name} disabled={editing.made} onChange={e => change({name: e.target.value})}/>
                </label>
                {FIELDS.map(({key, label}) => (
                    <label key={key} className="field">
                        <span className="muted">{label}</span>
                        <input
                            type="number"
                            inputMode="decimal"
                            value={draft[key]}
                            onChange={e => change({[key]: e.target.value})}
                        />
                    </label>
                ))}
            </div>
            <ConfirmAction
                label="Save"
                disabled={!complete(draft)}
                question={`Save ${draft.name.trim()}?`}
                run={() => saveRuleSet(planet, draft)}
                said={() => "Saved."}
                refusal={wordsOf}
                onDone={onSaved}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
