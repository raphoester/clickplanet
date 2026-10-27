import {useEffect, useRef, useState} from "react"
import {EndedSeason, HallOfFameBackend} from "../../backends/hallOfFame.ts"
import {holds} from "../../domain/hallOfFame.ts"

// The read is cached for a minute, so a season just ended can take that long to be in it.
export const HALL_RETRY_MS = 20_000
export const HALL_RETRIES = 6

export function useHallOfFame(backend: HallOfFameBackend | undefined, ended: number | undefined): EndedSeason[] | undefined {
    const [hall, setHall] = useState<EndedSeason[]>()
    const latest = useRef(hall)
    latest.current = hall

    useEffect(() => {
        if (!backend) return
        if (ended !== undefined && latest.current && holds(latest.current, ended)) return

        let stale = false
        let timer: ReturnType<typeof setTimeout> | undefined
        let retries = ended === undefined ? 0 : HALL_RETRIES
        const read = () => backend.hallOfFame().then(
            (found) => {
                if (stale) return
                setHall(found)
                if (ended !== undefined && !holds(found, ended) && retries-- > 0) timer = setTimeout(read, HALL_RETRY_MS)
            },
            (e) => console.error("Could not read the Hall of Fame", e),
        )
        void read()
        return () => {
            stale = true
            clearTimeout(timer)
        }
    }, [backend, ended])

    return hall
}
