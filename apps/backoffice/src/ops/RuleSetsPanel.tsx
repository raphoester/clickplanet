import {useState} from "react"
import {ChangeList} from "../components/Changes.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {wordsOf} from "../refusal.ts"
import {changesWord} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {Book, changesOf, complete, copyOf, Draft, draftOf, Kept, playedOf} from "./ruleBook.ts"
import {SwitchField, ValueField} from "./RuleFields.tsx"
import {DEFAULT_SET} from "./rules.ts"
import {changedAt, Switch} from "./ruleValues.ts"

// A column of the list of sets: what a set reads as there, as it plays.
export type Column<S> = {title: string, of: (played: S) => string}

// A module's answer to GetRules: every set it keeps, and the one it plays.
export type Sets<S> = {sets: S[], playing: string}

// made: an edit of a set kept, with its name fixed; otherwise a new set copied from `from`. defaults: default's values,
// for a set that plays them where it holds none.
type Editing = {draft: Draft, made: boolean, from: string, playing: string, defaults?: Draft}

type Props<S extends Kept> = {
    title: string
    hint: string
    book: Book<S>
    columns: Column<S>[]
    // Must keep its identity (module scope or useCallback), as useCall's.
    read: (signal: AbortSignal) => Promise<Sets<S>>
    save: (draft: Draft) => Promise<void>
    onOpen: (open: boolean) => void
}

// Every rule set a module keeps, and a form to make one or change one. `onOpen` says when the form opens and closes.
export function RuleSetsPanel<S extends Kept>({title, hint, book, columns, read, save, onOpen}: Props<S>) {
    const [rules, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const open = (opened: Editing | undefined) => {
        setEditing(opened)
        onOpen(opened !== undefined)
    }
    const saved = () => {
        open(undefined)
        again()
    }

    return (
        <section aria-label={title} className="detail">
            <h2 className="list-title">{title}</h2>
            {editing !== undefined && (
                <SetForm
                    key={`${editing.made ? "edit" : "copy"} ${editing.from}`}
                    book={book}
                    save={save}
                    editing={editing}
                    onCancel={() => open(undefined)}
                    onSaved={saved}
                />
            )}
            {editing === undefined && rules.state === "loading" && <p className="muted">Loading…</p>}
            {editing === undefined && rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {editing === undefined && rules.state === "done" && (
                <>
                    <p className="hint">Playing: <code>{rules.value.playing}</code></p>
                    <p className="hint">{hint}</p>
                    <SetList
                        book={book}
                        columns={columns}
                        rules={rules.value}
                        onEdit={set => open({
                            draft: draftOf(book, set), made: true, from: set.name, playing: rules.value.playing,
                            defaults: set.name === DEFAULT_SET ? undefined : defaultsOf(book, rules.value),
                        })}
                        onCopy={set => open({
                            draft: copyOf(book, set), made: false, from: set.name, playing: rules.value.playing,
                            defaults: defaultsOf(book, rules.value),
                        })}
                    />
                </>
            )}
        </section>
    )
}

function defaultsOf<S extends Kept>(book: Book<S>, rules: Sets<S>): Draft {
    return draftOf(book, rules.sets.find(set => set.name === DEFAULT_SET) ?? {name: DEFAULT_SET})
}

// Each set as it plays: its own values, and default's for the rest.
function SetList<S extends Kept>({book, columns, rules, onEdit, onCopy}: {
    book: Book<S>, columns: Column<S>[], rules: Sets<S>, onEdit: (set: S) => void, onCopy: (set: S) => void,
}) {
    const defaults = rules.sets.find(set => set.name === DEFAULT_SET)
    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr>
                    <th>Set</th>
                    <th/>
                    {columns.map(column => <th key={column.title}>{column.title}</th>)}
                    <th/>
                </tr>
            </thead>
            <tbody>
                {rules.sets.map(set => ({set, played: playedOf(book, set, defaults)})).map(({set, played}) => (
                    <tr key={set.name} aria-current={set.name === rules.playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{set.name === rules.playing && <span className="status status-good">Playing now</span>}</td>
                        {columns.map(column => <td key={column.title}>{column.of(played)}</td>)}
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

function SetForm<S extends Kept>({book, save, editing, onCancel, onSaved}: {
    book: Book<S>, save: (draft: Draft) => Promise<void>, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const value = (path: string, typed: string) => setDraft(was => ({...was, values: {...was.values, [path]: typed}}))
    const title = editing.made ? `Edit ${editing.from}` : `New set from ${editing.from}`
    const changes = changesOf(book, editing.draft, draft)
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
            <h3 className="list-title">{title}</h3>
            <div className="panel">
                <label className="field">
                    <span className="muted">Name</span>
                    <input value={draft.name} disabled={editing.made} onChange={e => setDraft(was => ({...was, name: e.target.value}))}/>
                </label>
                {defaults !== undefined && <p className="hint">An empty value plays default's.</p>}
                {switchOf(book.frozen)}
            </div>
            <div className="rule-groups">
                {book.groups.map(group => (
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
                    disabled={!complete(book, draft) || (editing.made && changes.length === 0)}
                    question={(
                        <>
                            <p>Save {draft.name.trim()}{changes.length === 0 ? `, as ${editing.from} is?` : "?"}</p>
                            {draft.name.trim() === editing.playing && <p>It plays now: it takes over at once.</p>}
                            {changes.length > 0 && <ChangeList changes={changes}/>}
                        </>
                    )}
                    run={() => save(draft)}
                    said={() => "Saved."}
                    refusal={wordsOf}
                    onDone={onSaved}
                />
                <button type="button" className="button" onClick={onCancel}>Cancel</button>
            </div>
        </section>
    )
}
