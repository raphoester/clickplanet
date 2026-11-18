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

type BoardProps = {
    label: string, board: string | undefined, defaults: string | undefined, inherits: boolean,
    // The boards the page carries, by name.
    boards: string[],
    onChange: (board: string | undefined) => void,
}

const WORLD = "world"
const DEFAULT = "default"

// The board an arena's fresh map plays: the world's, or one the page carries, by name; one the set holds that the page
// does not carry is listed too. Outside default, default's as well; in default, the world's is holding none.
export function BoardField({label, board, defaults, inherits, boards, onChange}: BoardProps) {
    const names = board === undefined || board === "" || boards.includes(board) ? boards : [...boards, board]
    const value = board === undefined ? (inherits ? DEFAULT : WORLD) : board === "" ? WORLD : `board:${board}`
    const picked = (option: string): string | undefined => {
        if (option === DEFAULT) return undefined
        if (option === WORLD) return inherits ? "" : undefined
        return option.slice("board:".length)
    }
    return (
        <div className="field">
            <span className="muted">{label}</span>
            <select aria-label={label} value={value} onChange={e => onChange(picked(e.target.value))}>
                {inherits && <option value={DEFAULT}>Default ({defaults === undefined || defaults === "" ? "the world's" : defaults})</option>}
                <option value={WORLD}>The world's</option>
                {names.map(name => <option key={name} value={`board:${name}`}>{name}</option>)}
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
