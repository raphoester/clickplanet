import {useCallback, useEffect, useState} from "react"
import {Failure, failureOf} from "../refusal.ts"

// The last answer stays while the next is read, and after a failure: a refresh never empties the page.
export type Polled<T> = {value: T | undefined, failure: Failure | undefined, reading: boolean}

// Runs `read` now and every `every` ms while mounted, never twice at once. `read` must keep its identity
// (module scope or useCallback), or it starts again on every render.
export function usePolling<T>(read: (signal: AbortSignal) => Promise<T>, every: number): [Polled<T>, () => void] {
    const [polled, setPolled] = useState<Polled<T>>({value: undefined, failure: undefined, reading: true})
    const [attempt, setAttempt] = useState(0)

    useEffect(() => {
        const controller = new AbortController()
        let busy = false

        const run = () => {
            if (busy) return
            busy = true
            setPolled(last => ({...last, reading: true}))
            read(controller.signal).then(
                value => {
                    if (!controller.signal.aborted) setPolled({value, failure: undefined, reading: false})
                },
                (err: unknown) => {
                    if (!controller.signal.aborted) setPolled(last => ({...last, failure: failureOf(err), reading: false}))
                },
            ).finally(() => {
                busy = false
            })
        }

        run()
        const timer = setInterval(run, every)
        return () => {
            controller.abort()
            clearInterval(timer)
        }
    }, [read, every, attempt])

    const again = useCallback(() => setAttempt(n => n + 1), [])
    return [polled, again]
}
