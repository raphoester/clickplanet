import {Field, shownOf, yesOf} from "./ruleValues.ts"

type SwitchProps = {
    label: string, on: boolean | undefined, defaults: boolean | undefined, inherits: boolean,
    onChange: (on: boolean | undefined) => void,
}

// In default, a box; in another set, its own yes or no, or default's.
export function SwitchField({label, on, defaults, inherits, onChange}: SwitchProps) {
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
export function ValueField({field, typed, inherited, was, onChange}: ValueProps) {
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
