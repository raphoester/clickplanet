import {useCallback, useState} from "react"

export type Step<T> =
    | {step: "idle"}
    | {step: "running"}
    | {step: "done", value: T}
    | {step: "refused", words: string}

// One read the operator starts by hand (a dry run, an inspection, a download), and what came of it. Unlike
// useCall it waits for a click; unlike ConfirmAction it asks nothing, since it changes nothing.
export function useAction<T>(refusal: (err: unknown) => string): [Step<T>, (run: () => Promise<T>) => void, () => void] {
    const [step, setStep] = useState<Step<T>>({step: "idle"})

    const start = useCallback((run: () => Promise<T>) => {
        setStep({step: "running"})
        run().then(
            value => setStep({step: "done", value}),
            (err: unknown) => setStep({step: "refused", words: refusal(err)}),
        )
    }, [refusal])

    const reset = useCallback(() => setStep({step: "idle"}), [])
    return [step, start, reset]
}
