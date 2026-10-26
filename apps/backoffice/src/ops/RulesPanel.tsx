import {useCallback, useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {boxesOf, clicksOf, complete, DEFAULT_SET, Draft, draftOf, GROUPS, saveRuleSet, tempoOf, TollStep} from "./rules.ts"

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
                <>
                    <Sets rules={rules.value} onEdit={set => setEditing({draft: draftOf(set), made: true})}/>
                    <NewSet rules={rules.value} onNew={draft => setEditing({draft, made: false})}/>
                </>
            )}
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

function Sets({rules, onEdit}: {rules: GetRulesResponse, onEdit: (set: RuleSet) => void}) {
    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr><th>Set</th><th>Refill</th><th>Clicks</th><th>Boxes</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === DEFAULT_SET ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{tempoOf(set)}</td>
                        <td>{clicksOf(set.clicks)}</td>
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

function NewSet({rules, onNew}: {rules: GetRulesResponse, onNew: (draft: Draft) => void}) {
    const played = rules.sets.find(set => set.name === DEFAULT_SET)
    if (played === undefined) return null

    return (
        <button type="button" className="button" onClick={() => onNew(draftOf(played, ""))}>
            New set
        </button>
    )
}

function SetForm({planet, editing, onCancel, onSaved}: {
    planet: PlanetAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const value = (path: string, typed: string) => setDraft(was => ({...was, values: {...was.values, [path]: typed}}))

    return (
        <section className="panel" aria-label={editing.made ? `Edit ${editing.draft.name}` : "New set"}>
            <h2>{editing.made ? `Edit ${editing.draft.name}` : "New set"}</h2>
            <label className="field">
                <span className="muted">Name</span>
                <input value={draft.name} disabled={editing.made} onChange={e => change({name: e.target.value})}/>
            </label>
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
                        {group.fields.map(({path, label}) => (
                            <label key={path} className="field">
                                <span className="muted">{label}</span>
                                <input
                                    type="number"
                                    inputMode="decimal"
                                    value={draft.values[path]}
                                    onChange={e => value(path, e.target.value)}
                                />
                            </label>
                        ))}
                    </div>
                </fieldset>
            ))}
            <Toll steps={draft.toll} onChange={toll => change({toll})}/>
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
