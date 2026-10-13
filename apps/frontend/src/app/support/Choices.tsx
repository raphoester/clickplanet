import {useId} from "react"

export type Choice<Value extends string> = {value: Value, label: string}

export type ChoicesProps<Value extends string> = {
    label: string
    options: readonly Choice<Value>[]
    picked?: Value
    onPick: (value: Value) => void
}

export default function Choices<Value extends string>({label, options, picked, onPick}: ChoicesProps<Value>) {
    const labelId = useId()

    return <div className="support-choices">
        <span className="menu-label" id={labelId}>{label}</span>
        <div className="support-choice-options" role="radiogroup" aria-labelledby={labelId}>
            {options.map((option) => <button key={option.value}
                                             type="button"
                                             role="radio"
                                             aria-checked={option.value === picked}
                                             className="panel-box support-choice"
                                             onClick={() => onPick(option.value)}>
                {option.label}
            </button>)}
        </div>
    </div>
}
