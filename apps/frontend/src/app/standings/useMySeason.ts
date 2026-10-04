import {useEffect, useRef, useState} from "react"
import {NameColor} from "../../backends/player.ts"
import {MySeason, StandingsBackend} from "../../backends/standings.ts"
import {liveSeason, Takes, takesSince} from "../../domain/standings.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useOwnTakes, useReadsAfterClicks} from "../viewer/useAcceptedClicks.ts"

export type Caller = {
    username?: string
    color?: NameColor
    linked: boolean
}

type Read = {
    countryCode: string
    season: MySeason
    before: Takes
}

export function useMySeason(
    backend: StandingsBackend,
    caller: Caller,
    listenForClicks: ListenForClicks,
    countryCode: string,
): MySeason | undefined {
    const takes = useOwnTakes(listenForClicks)
    const reads = useReadsAfterClicks(listenForClicks)
    const latest = useRef(takes)
    const [read, setRead] = useState<Read>()

    useEffect(() => {
        latest.current = takes
    }, [takes])

    useEffect(() => {
        let stale = false
        const before = latest.current
        backend.mySeason(countryCode).then(
            (season) => {
                if (!stale) setRead(season && {countryCode, season, before})
            },
            (e) => console.error("Could not read your season", e),
        )
        return () => {
            stale = true
        }
    }, [backend, caller.linked, caller.username, countryCode, reads])

    return read?.countryCode === countryCode ? liveSeason(read.season, takesSince(read.before, takes)) : undefined
}
