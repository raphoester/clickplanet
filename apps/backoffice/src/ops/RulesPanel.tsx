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
    boxesOf, changedAt, changesOf, clicksOf, complete, copyOf, DEFAULT_SET, Draft, draftOf, Field, GROUPS, playedOf, playOf,
    saveRuleSet, shownOf, tempoOf, tollOf, TollStep, yesOf,
} from "./rules.ts"

// made: an edit of a set kept, with its name fixed; otherwise a new set copied from `from`. Either way the form
// shows what it changes from that set. defaults: default's values, for a set that plays them where it holds none.
type Editing = {draft: Draft, made: boolean, from: string, playing: string, defaults?: Draft}

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
                        onEdit={set => setEditing({
                            draft: draftOf(set), made: true, from: set.name, playing: rules.value.playing,
                            defaults: set.name === DEFAULT_SET ? undefined : defaultsOf(rules.value),
                        })}
                        onCopy={set => setEditing({
                            draft: copyOf(set), made: false, from: set.name, playing: rules.value.playing,
                            defaults: defaultsOf(rules.value),
                        })}
                    />
                </>
            )}
        </section>
    )
}

function defaultsOf(rules: GetRulesResponse): Draft {
    return draftOf(rules.sets.find(set => set.name === DEFAULT_SET) ?? new RuleSet())
}

// Each set as it plays: its own values, and default's for the rest.
function Sets({rules, onEdit, onCopy}: {
    rules: GetRulesResponse, onEdit: (set: RuleSet) => void, onCopy: (set: RuleSet) => void,
}) {
    const defaults = rules.sets.find(set => set.name === DEFAULT_SET)
    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr><th>Set</th><th/><th>Map</th><th>Refill</th><th>Clicks</th><th>Boxes</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => ({set, played: playedOf(set, defaults)})).map(({set, played}) => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{set.name === rules.playing && <span className="status status-good">Playing now</span>}</td>
                        <td>{playOf(played)}</td>
                        <td>{tempoOf(played)}</td>
                        <td>{clicksOf(played.rules?.clicks)}</td>
                        <td>{boxesOf(played.rules?.bonuses?.boxes)}</td>
                        <td className="row-buttons">
                            <button type="button" className="button-small" aria-label={`Edit ${set.name}`} onClick={() => onEdit(set)}>
                                Edit
                            </button>
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
    const defaults = editing.defaults

    return (
        <section className="detail editor" aria-label={title}>
            <button type="button" className="button-small back" onClick={onCancel}>← All sets</button>
            <h2 className="list-title">{title}</h2>
            <div className="panel">
                <label className="field">
                    <span className="muted">Name</span>
                    <input value={draft.name} disabled={editing.made} onChange={e => change({name: e.target.value})}/>
                </label>
                {defaults !== undefined && <p className="hint">An empty value plays default's.</p>}
                <SwitchField label="Frozen: the map takes no play" on={draft.frozen} defaults={defaults?.frozen}
                             inherits={defaults !== undefined} onChange={frozen => change({frozen})}/>
                <SwitchField label="Flags locked: no player changes country" on={draft.allegianceLocked}
                             defaults={defaults?.allegianceLocked} inherits={defaults !== undefined}
                             onChange={allegianceLocked => change({allegianceLocked})}/>
            </div>
            <div className="rule-groups">
                {GROUPS.map(group => (
                    <fieldset key={group.title} className="rule-group">
                        <legend>{group.title}</legend>
                        {group.title === "New accounts" && (
                            <SwitchField label="Refill at first sign-in" on={draft.signInRefill} defaults={defaults?.signInRefill}
                                         inherits={defaults !== undefined} onChange={signInRefill => change({signInRefill})}/>
                        )}
                        {group.title === "Quizzes" && (
                            <SwitchField label="Quizzes on" on={draft.quiz} defaults={defaults?.quiz}
                                         inherits={defaults !== undefined} onChange={quiz => change({quiz})}/>
                        )}
                        <div className="fields">
                            {group.fields.map(field => (
                                <ValueField
                                    key={field.path}
                                    field={field}
                                    typed={draft.values[field.path] ?? ""}
                                    inherited={defaults === undefined ? undefined : defaults.values[field.path] ?? ""}
                                    was={changedAt(editing.draft, draft, field.path) ? editing.draft.values[field.path] : undefined}
                                    onChange={typed => value(field.path, typed)}
                                />
                            ))}
                        </div>
                    </fieldset>
                ))}
                <Toll steps={draft.toll} defaults={defaults?.toll} inherits={defaults !== undefined} onChange={toll => change({toll})}/>
            </div>
            <div className="save-bar">
                <span className={changes.length === 0 ? "muted" : "save-count"}>{changesWord(changes.length)}</span>
                <ConfirmAction
                    label="Save"
                    disabled={!complete(draft) || (editing.made && changes.length === 0)}
                    question={(
                        <>
                            <p>Save {draft.name.trim()}{changes.length === 0 ? `, as ${editing.from} is?` : "?"}</p>
                            {draft.name.trim() === editing.playing && <p>It plays now: it takes over at once.</p>}
                            {changes.length > 0 && <ChangeList changes={changes}/>}
                        </>
                    )}
                    run={() => saveRuleSet(planet, draft)}
                    said={() => "Saved."}
                    refusal={wordsOf}
                    onDone={onSaved}
                />
                <button type="button" className="button" onClick={onCancel}>Cancel</button>
            </div>
        </section>
    )
}

type SwitchProps = {
    label: string, on: boolean | undefined, defaults: boolean | undefined, inherits: boolean,
    onChange: (on: boolean | undefined) => void,
}

// In default, a box; in another set, its own yes or no, or default's.
function SwitchField({label, on, defaults, inherits, onChange}: SwitchProps) {
    if (!inherits) {
        return (
            <label className="check">
                <input type="checkbox" checked={on === true} onChange={e => onChange(e.target.checked)}/>
                {label}
            </label>
        )
    }
    return (
        <div className="field">
            <span className="muted">{label}</span>
            <select aria-label={label} value={yesOf(on)}
                    onChange={e => onChange(e.target.value === "default" ? undefined : e.target.value === "yes")}>
                <option value="default">Default ({yesOf(defaults)})</option>
                <option value="yes">Yes</option>
                <option value="no">No</option>
            </select>
        </div>
    )
}

type ValueProps = {
    field: Field, typed: string, inherited: string | undefined, was: string | undefined, onChange: (typed: string) => void,
}

// A value of the set: a duration says what it reads as, and a value changed says what it was. Outside default, an
// empty value plays default's, which it shows.
function ValueField({field, typed, inherited, was, onChange}: ValueProps) {
    const shown = shownOf(field, typed)
    const follows = inherited !== undefined && typed.trim() === ""
    const notes = [
        ...(follows ? [`default: ${shownOf(field, inherited)}`] : []),
        ...(shown !== typed && shown !== `${typed.trim()} s` && shown !== "" ? [shown] : []),
        ...(was !== undefined ? [`was ${shownOf(field, was) || (inherited === undefined ? "empty" : "default")}`] : []),
    ]
    return (
        <div className={was === undefined ? "field" : "field field-changed"}>
            <label className="field">
                <span className="muted">{field.label}</span>
                <input type="number" inputMode="decimal" value={typed} placeholder={inherited}
                       onChange={e => onChange(e.target.value)}/>
            </label>
            {notes.length > 0 && <span className="hint">{notes.join(" · ")}</span>}
            {inherited !== undefined && !follows && (
                <button type="button" className="button-mini" aria-label={`${field.label}: use default`} onClick={() => onChange("")}>
                    Use default
                </button>
            )}
        </div>
    )
}

type TollProps = {
    steps: TollStep[] | undefined, defaults: TollStep[] | undefined, inherits: boolean,
    onChange: (steps: TollStep[] | undefined) => void,
}

// Outside default, the toll is default's until the set holds one of its own, every step of it.
function Toll({steps, defaults, inherits, onChange}: TollProps) {
    const edit = (at: number, part: Partial<TollStep>) =>
        onChange((steps ?? []).map((step, i) => i === at ? {...step, ...part} : step))

    return (
        <fieldset className="rule-group">
            <legend>Toll</legend>
            {inherits && (
                <label className="check">
                    <input type="checkbox" checked={steps !== undefined}
                           onChange={e => onChange(e.target.checked ? [...(defaults ?? [])] : undefined)}/>
                    A toll of its own
                </label>
            )}
            {steps === undefined && <p className="hint">Default's: {tollOf(defaults)}</p>}
            {steps?.map((step, i) => (
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
            {steps !== undefined && (
                <button type="button" className="button-mini" onClick={() => onChange([...steps, {share: "", slowdown: ""}])}>
                    Add a step
                </button>
            )}
        </fieldset>
    )
}
