import {useEffect, useState} from "react"
import {Season, SeasonBackend} from "../../backends/season.ts"

// A longer setTimeout delay overflows and fires at once.
const LONGEST_TIMEOUT_MS = 2 ** 31 - 1

export function useSeason(backend: SeasonBackend | undefined): Season | undefined {
    const [season, setSeason] = useState<Season>()

    useEffect(() => {
        if (!backend) return

        let stale = false
        backend.season().then(
            (read) => {
                if (!stale) setSeason(read)
            },
            (e) => console.error("Could not read the season", e),
        )
        return () => {
            stale = true
        }
    }, [backend])

    useEffect(() => {
        if (!season) return

        let timer: ReturnType<typeof setTimeout> | undefined
        const waitForTheEnd = () => {
            const left = season.endsAt - Date.now()
            if (left <= 0) {
                setSeason(undefined)
                return
            }
            timer = setTimeout(waitForTheEnd, Math.min(left, LONGEST_TIMEOUT_MS))
        }
        waitForTheEnd()
        return () => clearTimeout(timer)
    }, [season])

    return season
}
