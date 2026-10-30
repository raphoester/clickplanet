import {CSSProperties, useId} from "react"
import "./Choices.css"

export type Choice<Value extends string> = {value: Value, label: string}

export type ChoicesProps<Value extends string> = {
    label: string
    options: readonly Choice<Value>[]
    picked?: Value
    onPick: (value: Value) => void
}

// Up to three in a row; four go in two rows of two.
const MOST_IN_A_ROW = 3

export default function Choices<Value extends string>({label, options, picked, onPick}: ChoicesProps<Value>) {
    const labelId = useId()
    const columns = options.length <= MOST_IN_A_ROW ? options.length : 2

    return <div className="choices">
        <span className="menu-label" id={labelId}>{label}</span>
        <div className="choices-options"
             role="radiogroup"
             aria-labelledby={labelId}
             style={{"--choices-columns": columns} as CSSProperties}>
            {options.map((option) => <button key={option.value}
                                             type="button"
                                             role="radio"
                                             aria-checked={option.value === picked}
                                             className="panel-box choice"
                                             onClick={() => onPick(option.value)}>
                {option.label}
            </button>)}
        </div>
    </div>
}
