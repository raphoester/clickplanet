import {useEffect, useState} from "react"
import {NO_TAKES, Takes, withTake} from "../../domain/standings.ts"
import {ListenForClicks} from "./acceptedClicks.ts"

export const SETTLE_MS = 2_000

export const LONGEST_MS = 10_000

export function useReadsAfterClicks(listenForClicks: ListenForClicks | undefined): number {
    const [reads, setReads] = useState(0)

    useEffect(() => {
        if (!listenForClicks) return
        let settle: ReturnType<typeof setTimeout> | undefined
        let longest: ReturnType<typeof setTimeout> | undefined
        const read = () => {
            clearTimeout(settle)
            clearTimeout(longest)
            settle = undefined
            longest = undefined
            setReads((n) => n + 1)
        }
        const stop = listenForClicks(() => {
            clearTimeout(settle)
            settle = setTimeout(read, SETTLE_MS)
            longest ??= setTimeout(read, LONGEST_MS)
        })
        return () => {
            stop()
            clearTimeout(settle)
            clearTimeout(longest)
        }
    }, [listenForClicks])

    return reads
}

export function useOwnTakes(listenForClicks: ListenForClicks | undefined): Takes {
    const [takes, setTakes] = useState(NO_TAKES)

    useEffect(() => {
        if (!listenForClicks) return
        return listenForClicks((click) => {
            if (click.took) setTakes((current) => withTake(current, click.country))
        })
    }, [listenForClicks])

    return takes
}
