import {useEffect, useState} from "react"
import {Season, SeasonBackend} from "../../backends/season.ts"

// A longer setTimeout delay overflows and fires at once.
const LONGEST_TIMEOUT_MS = 2 ** 31 - 1

export type Seasons = {
    season?: Season
    next?: Season
}

export function useSeason(backend: SeasonBackend | undefined): Seasons {
    const [read, setRead] = useState<Seasons>({})
    const [, setChanges] = useState(0)

    useEffect(() => {
        if (!backend) return

        let stale = false
        Promise.all([backend.season(), backend.next()]).then(
            ([season, next]) => {
                if (!stale) setRead({season, next})
            },
            (e) => console.error("Could not read the season", e),
        )
        return () => {
            stale = true
        }
    }, [backend])

    const seasons = seasonsAt(read, Date.now())
    const change = changeOf(seasons, Date.now())

    useEffect(() => {
        if (change === undefined) return

        let timer: ReturnType<typeof setTimeout> | undefined
        const waitForTheChange = () => {
            const left = change - Date.now()
            if (left <= 0) {
                setChanges((changes) => changes + 1)
                return
            }
            timer = setTimeout(waitForTheChange, Math.min(left, LONGEST_TIMEOUT_MS))
        }
        waitForTheChange()
        return () => clearTimeout(timer)
    }, [change])

    return seasons
}

function seasonsAt({season, next}: Seasons, now: number): Seasons {
    if (next && (next.startsAt ?? 0) <= now) return {season: now < next.endsAt ? next : undefined}
    return {season: season && now < season.endsAt ? season : undefined, next}
}

function changeOf({season, next}: Seasons, now: number): number | undefined {
    const changes = [season?.endsAt, next?.startsAt].filter((at): at is number => at !== undefined && at > now)
    return changes.length > 0 ? Math.min(...changes) : undefined
}
