import {useId} from "react";
import "./Switch.css"

export type SwitchProps = {
    label: string
    checked: boolean
    disabled?: boolean
    main?: boolean
    onChange: (checked: boolean) => void
}

export default function Switch({label, checked, disabled, main, onChange}: SwitchProps) {
    const id = useId()
    return <label className={main ? "switch switch--main panel-box" : "switch"} htmlFor={id}>
        <span className="switch-label">{label}</span>
        <input id={id}
               type="checkbox"
               role="switch"
               className="switch-input"
               checked={checked}
               disabled={disabled}
               onChange={(event) => onChange(event.target.checked)}/>
    </label>
}
