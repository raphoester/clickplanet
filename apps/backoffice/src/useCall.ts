import {useCallback, useEffect, useState} from "react"
import {Failure, failureOf} from "./refusal.ts"

export type Call<T> =
    | {state: "loading"}
    | {state: "done", value: T}
    | ({state: "failed"} & Failure)

// `run` must keep its identity between renders (module scope or useCallback), or it runs on every render.
export function useCall<T>(run: (signal: AbortSignal) => Promise<T>): [Call<T>, () => void] {
    const [call, setCall] = useState<Call<T>>({state: "loading"})
    const [attempt, setAttempt] = useState(0)

    useEffect(() => {
        const controller = new AbortController()
        setCall({state: "loading"})
        run(controller.signal).then(
            value => {
                if (!controller.signal.aborted) setCall({state: "done", value})
            },
            (err: unknown) => {
                if (!controller.signal.aborted) setCall({state: "failed", ...failureOf(err)})
            },
        )
        return () => controller.abort()
    }, [run, attempt])

    const again = useCallback(() => setAttempt(n => n + 1), [])
    return [call, again]
}

// What the call answered last, kept on screen while it runs again.
export function useLast<T>(call: Call<T>): T | undefined {
    const [last, setLast] = useState<T | undefined>(undefined)

    useEffect(() => {
        if (call.state === "done") setLast(call.value)
    }, [call])

    return call.state === "done" ? call.value : last
}
