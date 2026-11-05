import {useEffect, useState} from "react"
import {Season} from "../../backends/season.ts"

// A longer setTimeout delay overflows and fires at once.
const LONGEST_TIMEOUT_MS = 2 ** 31 - 1

// The season the timeline says runs, let go at its end on this clock.
export function useSeason(current: Season | undefined): Season | undefined {
    const [ended, setEnded] = useState<Season>()

    useEffect(() => {
        if (!current) return

        let timer: ReturnType<typeof setTimeout> | undefined
        const waitForTheEnd = () => {
            const left = current.endsAt - Date.now()
            if (left <= 0) {
                setEnded(current)
                return
            }
            timer = setTimeout(waitForTheEnd, Math.min(left, LONGEST_TIMEOUT_MS))
        }
        waitForTheEnd()
        return () => clearTimeout(timer)
    }, [current])

    return current && current !== ended ? current : undefined
}
