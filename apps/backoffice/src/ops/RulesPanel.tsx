import {useCallback, useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ChangeList} from "../components/Changes.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {changesWord} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {
    boxesOf, changedAt, changesOf, clicksOf, complete, copyOf, Draft, draftOf, Field, GROUPS, playOf, saveRuleSet, shownOf,
    tempoOf, TollStep,
} from "./rules.ts"

// made: an edit of a set kept, with its name fixed; otherwise a new set copied from `from`. Either way the form
// shows what it changes from that set.
type Editing = {draft: Draft, made: boolean, from: string}

const refusal = (err: unknown) =>
    wordsOf(err, {failedPrecondition: "A set of this name plays now. Save it under another name."})

// Every rule set postgres keeps, and a form to make one or change one.
export function RulesPanel({planet}: {planet: PlanetAdmin}) {
    const read = useCallback((signal: AbortSignal) => planet.getRules({}, {signal}), [planet])
    const [rules, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const saved = () => {
        setEditing(undefined)
        again()
    }

    if (editing !== undefined) {
        return (
            <SetForm
                key={`${editing.made ? "edit" : "copy"} ${editing.from}`}
                planet={planet}
                editing={editing}
                onCancel={() => setEditing(undefined)}
                onSaved={saved}
            />
        )
    }

    return (
        <section aria-label="Game rules" className="detail">
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && (
                <>
                    <p className="hint">Playing: <code>{rules.value.playing}</code></p>
                    <Sets
                        rules={rules.value}
                        onEdit={set => setEditing({draft: draftOf(set), made: true, from: set.name})}
                        onCopy={set => setEditing({draft: copyOf(set), made: false, from: set.name})}
                    />
                </>
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
                <tr><th>Set</th><th/><th>Map</th><th>Refill</th><th>Clicks</th><th>Boxes</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{set.name === rules.playing && <span className="status status-good">Playing now</span>}</td>
                        <td>{playOf(set)}</td>
                        <td>{tempoOf(set)}</td>
                        <td>{clicksOf(set.rules?.clicks)}</td>
                        <td>{boxesOf(set.rules?.bonuses?.boxes)}</td>
                        <td className="row-buttons">
                            {set.name !== rules.playing && (
                                <button type="button" className="button-small" aria-label={`Edit ${set.name}`} onClick={() => onEdit(set)}>
                                    Edit
                                </button>
                            )}
                            <button type="button" className="button-small" aria-label={`Copy ${set.name}`} onClick={() => onCopy(set)}>
                                Copy
                            </button>
                        </td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

function SetForm({planet, editing, onCancel, onSaved}: {
    planet: PlanetAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const value = (path: string, typed: string) => setDraft(was => ({...was, values: {...was.values, [path]: typed}}))
    const title = editing.made ? `Edit ${editing.from}` : `New set from ${editing.from}`
    const changes = changesOf(editing.draft, draft)

    return (
        <section className="detail editor" aria-label={title}>
            <button type="button" className="button-small back" onClick={onCancel}>← All sets</button>
            <h2 className="list-title">{title}</h2>
            <div className="panel">
                <label className="field">
                    <span className="muted">Name</span>
                    <input value={draft.name} disabled={editing.made} onChange={e => change({name: e.target.value})}/>
                </label>
                <label className="check">
                    <input type="checkbox" checked={draft.frozen} onChange={e => change({frozen: e.target.checked})}/>
                    Frozen: the map takes no play
                </label>
            </div>
            <div className="rule-groups">
                {GROUPS.map(group => (
                    <fieldset key={group.title} className="rule-group">
                        <legend>{group.title}</legend>
                        {group.title === "Quizzes" && (
                            <label className="check">
                                <input type="checkbox" checked={draft.quiz} onChange={e => change({quiz: e.target.checked})}/>
                                Quizzes on
                            </label>
                        )}
                        <div className="fields">
                            {group.fields.map(field => (
                                <ValueField
                                    key={field.path}
                                    field={field}
                                    typed={draft.values[field.path]}
                                    was={changedAt(editing.draft, draft, field.path) ? editing.draft.values[field.path] : undefined}
                                    onChange={typed => value(field.path, typed)}
                                />
                            ))}
                        </div>
                    </fieldset>
                ))}
                <Toll steps={draft.toll} onChange={toll => change({toll})}/>
            </div>
            <div className="save-bar">
                <span className={changes.length === 0 ? "muted" : "save-count"}>{changesWord(changes.length)}</span>
                <ConfirmAction
                    label="Save"
                    disabled={!complete(draft) || (editing.made && changes.length === 0)}
                    question={(
                        <>
                            <p>Save {draft.name.trim()}{changes.length === 0 ? `, as ${editing.from} is?` : "?"}</p>
                            {changes.length > 0 && <ChangeList changes={changes}/>}
                        </>
                    )}
                    run={() => saveRuleSet(planet, draft)}
                    said={() => "Saved."}
                    refusal={refusal}
                    onDone={onSaved}
                />
                <button type="button" className="button" onClick={onCancel}>Cancel</button>
            </div>
        </section>
    )
}

type ValueProps = {field: Field, typed: string, was: string | undefined, onChange: (typed: string) => void}

// A value of the set: a duration says what it reads as, and a value changed says what it was.
function ValueField({field, typed, was, onChange}: ValueProps) {
    const shown = shownOf(field, typed)
    const notes = [
        ...(shown !== typed && shown !== `${typed.trim()} s` && shown !== "" ? [shown] : []),
        ...(was !== undefined ? [`was ${shownOf(field, was) || "empty"}`] : []),
    ]
    return (
        <div className={was === undefined ? "field" : "field field-changed"}>
            <label className="field">
                <span className="muted">{field.label}</span>
                <input type="number" inputMode="decimal" value={typed} onChange={e => onChange(e.target.value)}/>
            </label>
            {notes.length > 0 && <span className="hint">{notes.join(" · ")}</span>}
        </div>
    )
}

function Toll({steps, onChange}: {steps: TollStep[], onChange: (steps: TollStep[]) => void}) {
    const edit = (at: number, part: Partial<TollStep>) =>
        onChange(steps.map((step, i) => i === at ? {...step, ...part} : step))

    return (
        <fieldset className="rule-group">
            <legend>Toll</legend>
            {steps.map((step, i) => (
                <div key={i} className="fields" role="group" aria-label={`Step ${i + 1}`}>
                    <label className="field">
                        <span className="muted">From % of the map</span>
                        <input type="number" inputMode="decimal" value={step.share}
                            onChange={e => edit(i, {share: e.target.value})}/>
                    </label>
                    <label className="field">
                        <span className="muted">Times slower</span>
                        <input type="number" inputMode="decimal" value={step.slowdown}
                            onChange={e => edit(i, {slowdown: e.target.value})}/>
                    </label>
                    <button type="button" className="button-mini" onClick={() => onChange(steps.filter((_, j) => j !== i))}>
                        Remove step {i + 1}
                    </button>
                </div>
            ))}
            <button type="button" className="button-mini" onClick={() => onChange([...steps, {share: "", slowdown: ""}])}>
                Add a step
            </button>
        </fieldset>
    )
}
