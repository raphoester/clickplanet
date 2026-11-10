import {CSSProperties, useEffect, useRef, useState} from "react";
import "./HoldButton.css"

type HoldButtonProps = {
    label: string;
    holdMs: number;
    disabled?: boolean;
    onHeld: () => void;
}

const holds = (key: string) => key === " " || key === "Enter"

export default function HoldButton({label, holdMs, disabled, onHeld}: HoldButtonProps) {
    const [holding, setHolding] = useState(false)
    const timer = useRef<number | undefined>(undefined)

    useEffect(() => () => window.clearTimeout(timer.current), [])

    const press = () => {
        if (disabled || timer.current !== undefined) return
        setHolding(true)
        timer.current = window.setTimeout(() => {
            timer.current = undefined
            setHolding(false)
            onHeld()
        }, holdMs)
    }

    const release = () => {
        window.clearTimeout(timer.current)
        timer.current = undefined
        setHolding(false)
    }

    return <div className="hold-button" style={{"--hold-ms": `${holdMs}ms`} as CSSProperties}>
        <button
            type="button"
            className="button button-action"
            disabled={disabled}
            onPointerDown={(e) => {
                if (e.button === 0) press()
            }}
            onPointerUp={release}
            onPointerLeave={release}
            onPointerCancel={release}
            onKeyDown={(e) => {
                if (!holds(e.key)) return
                e.preventDefault()
                if (!e.repeat) press()
            }}
            onKeyUp={(e) => {
                if (holds(e.key)) release()
            }}
            onBlur={release}
            onContextMenu={(e) => e.preventDefault()}>
            {label}
        </button>
        <div className="hold-button-track" aria-hidden="true">
            <div className={holding ? "hold-button-fill hold-button-filling" : "hold-button-fill"}/>
        </div>
    </div>
}
