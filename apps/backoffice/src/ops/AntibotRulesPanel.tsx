import {useCallback, useState} from "react"
import {AntibotAdmin} from "../clients.ts"
import {ChangeList} from "../components/Changes.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {GetRulesResponse, RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {changesWord} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {
    bansOf, changedAt, changesOf, copyOf, crowdOf, Draft, draftOf, flawOf, Group, Leaf, offOf, PARTS, placeOf, saveRuleSet,
    shownOf, Value,
} from "./antibotRules.ts"

// made: an edit of a set kept, with its name fixed; otherwise a new set copied from `from`.
type Editing = {draft: Draft, made: boolean, from: string}

const refusal = (err: unknown) =>
    wordsOf(err, {failedPrecondition: "A set of this name plays now. Save it under another name."})

// Every antibot rule set postgres keeps, and a form to make one or change one.
export function AntibotRulesPanel({antibot}: {antibot: AntibotAdmin}) {
    const read = useCallback((signal: AbortSignal) => antibot.getRules({}, {signal}), [antibot])
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
                antibot={antibot}
                editing={editing}
                onCancel={() => setEditing(undefined)}
                onSaved={saved}
            />
        )
    }

    return (
        <section aria-label="Anti-bot rules" className="detail">
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
                <tr><th>Set</th><th/><th>Bans</th><th>Watchdogs off</th><th>Crowd</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{set.name === rules.playing && <span className="status status-good">Playing now</span>}</td>
                        <td>{bansOf(set)}</td>
                        <td>{offOf(set)}</td>
                        <td>{crowdOf(set)}</td>
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

function SetForm({antibot, editing, onCancel, onSaved}: {
    antibot: AntibotAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const put = (path: string, value: Value) => setDraft(was => ({...was, values: {...was.values, [path]: value}}))
    const flaw = flawOf(draft)
    const changes = changesOf(editing.draft, draft)
    const title = editing.made ? `Edit ${editing.from}` : `New set from ${editing.from}`

    return (
        <section className="detail editor" aria-label={title}>
            <button type="button" className="button-small back" onClick={onCancel}>← All sets</button>
            <h2 className="list-title">{title}</h2>
            <div className="panel">
                <label className="field">
                    <span className="muted">Name</span>
                    <input
                        value={draft.name}
                        disabled={editing.made}
                        onChange={e => setDraft(was => ({...was, name: e.target.value}))}
                    />
                </label>
            </div>
            <div className="rule-parts">
                {PARTS.map(part => <Part key={part.path} group={part} draft={draft} before={editing.draft} onPut={put}/>)}
            </div>
            {flaw !== undefined && <p className="action-refused" role="alert">{flaw}</p>}
            <div className="save-bar">
                <span className={changes.length === 0 ? "muted" : "save-count"}>{changesWord(changes.length)}</span>
                <ConfirmAction
                    label="Save"
                    disabled={draft.name.trim() === "" || flaw !== undefined || (editing.made && changes.length === 0)}
                    question={(
                        <>
                            <p>Save {draft.name.trim()}{changes.length === 0 ? `, as ${editing.from} is?` : "?"}</p>
                            {changes.length > 0 && <ChangeList changes={changes}/>}
                        </>
                    )}
                    run={() => saveRuleSet(antibot, draft)}
                    said={() => "Saved."}
                    refusal={refusal}
                    onDone={onSaved}
                />
                <button type="button" className="button" onClick={onCancel}>Cancel</button>
            </div>
        </section>
    )
}

type PartProps = {group: Group, draft: Draft, before: Draft, onPut: (path: string, value: Value) => void}

// A watchdog, the bans or the jury, shut under one line that says whether it is on and what the form changed in it.
function Part({group, draft, before, onPut}: PartProps) {
    const toggle = group.leaves.find(leaf => leaf.kind === "switch")
    const changed = leavesIn(group).filter(leaf => changedAt(before, draft, leaf)).length
    return (
        <details className="rule-part">
            <summary>
                <span className="part-name">{group.label}</span>
                {toggle !== undefined && (
                    <span className={draft.values[toggle.path] === true ? "status status-good" : "status status-bad"}>
                        {toggle.label === "Enabled" ? "" : `${toggle.label}: `}{shownOf(toggle, draft.values[toggle.path])}
                    </span>
                )}
                {changed > 0 && <span className="status status-warn">{changesWord(changed)}</span>}
            </summary>
            <Fields group={group} draft={draft} before={before} onPut={onPut}/>
        </details>
    )
}

function Fields({group, draft, before, onPut}: PartProps) {
    return (
        <div className="part-body">
            <div className="fields">
                {group.leaves.map(leaf => (
                    <LeafField
                        key={leaf.path}
                        leaf={leaf}
                        value={draft.values[leaf.path]}
                        was={changedAt(before, draft, leaf) ? shownOf(leaf, before.values[leaf.path]) : undefined}
                        onPut={value => onPut(leaf.path, value)}
                    />
                ))}
            </div>
            {group.groups.map(inner => (
                <fieldset key={inner.path} className="rule-group">
                    <legend>{inner.label}</legend>
                    <Fields group={inner} draft={draft} before={before} onPut={onPut}/>
                </fieldset>
            ))}
        </div>
    )
}

function leavesIn(group: Group): Leaf[] {
    return [...group.leaves, ...group.groups.flatMap(leavesIn)]
}

type LeafProps = {leaf: Leaf, value: Value | undefined, was: string | undefined, onPut: (value: Value) => void}

// One value: a switch, a number, or a duration in seconds that says what it reads as. A value changed says what it was.
function LeafField({leaf, value, was, onPut}: LeafProps) {
    const className = was === undefined ? "field" : "field field-changed"
    const wasNote = was === undefined ? [] : [`was ${was === "" ? "unset" : was}`]
    switch (leaf.kind) {
        case "switch":
            return (
                <label className={was === undefined ? "check" : "check field-changed"}>
                    <input type="checkbox" aria-label={placeOf(leaf)} checked={value === true} onChange={e => onPut(e.target.checked)}/>
                    {leaf.label}
                </label>
            )
        case "number":
        case "whole":
        case "seconds": {
            const typed = typeof value === "string" ? value : ""
            const shown = shownOf(leaf, typed)
            const notes = [...(leaf.kind === "seconds" && shown !== `${typed.trim()} s` && shown !== "" ? [shown] : []), ...wasNote]
            return (
                <div className={className}>
                    <label className="field">
                        <span className="muted">{leaf.kind === "seconds" ? `${leaf.label} (s)` : leaf.label}</span>
                        <input
                            type="number"
                            inputMode="decimal"
                            step="any"
                            aria-label={placeOf(leaf)}
                            placeholder={leaf.optional ? "unset" : undefined}
                            value={typed}
                            onChange={e => onPut(e.target.value)}
                        />
                    </label>
                    {notes.length > 0 && <span className="hint">{notes.join(" · ")}</span>}
                </div>
            )
        }
        case "secondsList": {
            const steps = Array.isArray(value) ? value : []
            const edit = (at: number, typed: string) => onPut(steps.map((step, i) => i === at ? typed : step))
            return (
                <fieldset className={`rule-group steps ${was === undefined ? "" : "field-changed"}`}>
                    <legend>{leaf.label} (s)</legend>
                    {steps.map((step, i) => (
                        <div key={i} className="step">
                            <input
                                type="number"
                                inputMode="decimal"
                                step="any"
                                aria-label={`${placeOf(leaf)} ${i + 1}`}
                                value={step}
                                onChange={e => edit(i, e.target.value)}
                            />
                            <span className="hint">{shownOf(leaf, [step])}</span>
                            <button
                                type="button"
                                className="button-small"
                                aria-label={`Remove ${leaf.label.toLowerCase()} ${i + 1}`}
                                onClick={() => onPut(steps.filter((_, j) => j !== i))}
                            >
                                Remove
                            </button>
                        </div>
                    ))}
                    <button type="button" className="button-small" onClick={() => onPut([...steps, ""])}>Add a step</button>
                    {wasNote.length > 0 && <span className="hint">{wasNote[0]}</span>}
                </fieldset>
            )
        }
    }
}
