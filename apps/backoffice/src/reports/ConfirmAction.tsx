import {ReactNode, useState} from "react"

type Step<T> =
    | {step: "idle"}
    | {step: "asking"}
    | {step: "running"}
    | {step: "done", value: T}
    | {step: "refused", words: string}

type Props<T> = {
    label: string
    question: string
    run: () => Promise<T>
    said: (value: T) => ReactNode
    refusal: (err: unknown) => string
    onDone?: (value: T) => void
    disabled?: boolean
    tone?: "danger"
}

export function ConfirmAction<T>({label, question, run, said, refusal, onDone, disabled, tone}: Props<T>) {
    const [state, setState] = useState<Step<T>>({step: "idle"})

    const confirm = () => {
        setState({step: "running"})
        run().then(
            value => {
                setState({step: "done", value})
                onDone?.(value)
            },
            (err: unknown) => setState({step: "refused", words: refusal(err)}),
        )
    }

    if (state.step === "asking" || state.step === "running") {
        return (
            <div className="confirm" role="group" aria-label={label}>
                <p className="confirm-question">{question}</p>
                <div className="confirm-buttons">
                    <button
                        type="button"
                        className={`button ${tone === "danger" ? "button-danger" : "button-primary"}`}
                        disabled={state.step === "running"}
                        onClick={confirm}
                    >
                        {state.step === "running" ? "Working…" : `Yes, ${label.toLowerCase()}`}
                    </button>
                    <button
                        type="button"
                        className="button"
                        disabled={state.step === "running"}
                        onClick={() => setState({step: "idle"})}
                    >
                        Cancel
                    </button>
                </div>
            </div>
        )
    }

    return (
        <div className="action">
            <button
                type="button"
                className={`button ${tone === "danger" ? "button-danger-outline" : ""}`}
                disabled={disabled}
                onClick={() => setState({step: "asking"})}
            >
                {label}
            </button>
            {state.step === "done" && <p className="action-said" role="status">{said(state.value)}</p>}
            {state.step === "refused" && <p className="action-refused" role="alert">{state.words}</p>}
        </div>
    )
}
