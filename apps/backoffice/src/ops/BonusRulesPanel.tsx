import {useCallback, useState} from "react"
import {BonusesAdmin} from "../clients.ts"
import {ChangeList} from "../components/Changes.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse, RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {changesWord} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {
    boxesOf, changesOf, complete, copyOf, Draft, draftOf, FROZEN, GROUPS, offersOf, playedOf, quizzesOf, saveRuleSet,
} from "./bonusRules.ts"
import {SwitchField, ValueField} from "./RuleFields.tsx"
import {DEFAULT_SET} from "./rules.ts"
import {changedAt, Switch} from "./ruleValues.ts"

// made: an edit of a set kept, with its name fixed; otherwise a new set copied from `from`. defaults: default's values,
// for a set that plays them where it holds none.
type Editing = {draft: Draft, made: boolean, from: string, playing: string, defaults?: Draft}

// Every rule set the offers keep, and a form to make one or change one.
export function BonusRulesPanel({bonuses}: {bonuses: BonusesAdmin}) {
    const read = useCallback((signal: AbortSignal) => bonuses.getRules({}, {signal}), [bonuses])
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
                bonuses={bonuses}
                editing={editing}
                onCancel={() => setEditing(undefined)}
                onSaved={saved}
            />
        )
    }

    return (
        <section aria-label="Box & quiz rules" className="detail">
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && (
                <>
                    <p className="hint">Playing: <code>{rules.value.playing}</code></p>
                    <p className="hint">What a hand holds and what a charge does are the Game rules' Charges.</p>
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
                <tr><th>Set</th><th/><th>Offers</th><th>Boxes</th><th>Quizzes</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => ({set, played: playedOf(set, defaults)})).map(({set, played}) => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{set.name === rules.playing && <span className="status status-good">Playing now</span>}</td>
                        <td>{offersOf(played)}</td>
                        <td>{boxesOf(played.rules?.boxes)}</td>
                        <td>{quizzesOf(played.rules?.quiz)}</td>
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

function SetForm({bonuses, editing, onCancel, onSaved}: {
    bonuses: BonusesAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const value = (path: string, typed: string) => setDraft(was => ({...was, values: {...was.values, [path]: typed}}))
    const title = editing.made ? `Edit ${editing.from}` : `New set from ${editing.from}`
    const changes = changesOf(editing.draft, draft)
    const defaults = editing.defaults
    const switchOf = (each: Switch) => (
        <SwitchField
            key={each.path}
            label={each.label}
            on={draft.switches[each.path]}
            defaults={defaults?.switches[each.path]}
            inherits={defaults !== undefined}
            onChange={on => setDraft(was => ({...was, switches: {...was.switches, [each.path]: on}}))}
        />
    )

    return (
        <section className="detail editor" aria-label={title}>
            <button type="button" className="button-small back" onClick={onCancel}>← All sets</button>
            <h2 className="list-title">{title}</h2>
            <div className="panel">
                <label className="field">
                    <span className="muted">Name</span>
                    <input value={draft.name} disabled={editing.made} onChange={e => setDraft(was => ({...was, name: e.target.value}))}/>
                </label>
                {defaults !== undefined && <p className="hint">An empty value plays default's.</p>}
                {switchOf(FROZEN)}
            </div>
            <div className="rule-groups">
                {GROUPS.map(group => (
                    <fieldset key={group.title} className="rule-group">
                        <legend>{group.title}</legend>
                        {group.switches.map(switchOf)}
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
                    run={() => saveRuleSet(bonuses, draft)}
                    said={() => "Saved."}
                    refusal={wordsOf}
                    onDone={onSaved}
                />
                <button type="button" className="button" onClick={onCancel}>Cancel</button>
            </div>
        </section>
    )
}
