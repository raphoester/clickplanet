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
    season: MySeason
    before: Takes
}

export function useMySeason(backend: StandingsBackend, caller: Caller, listenForClicks: ListenForClicks): MySeason | undefined {
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
        backend.mySeason().then(
            (season) => {
                if (!stale) setRead(season && {season, before})
            },
            (e) => console.error("Could not read your season", e),
        )
        return () => {
            stale = true
        }
    }, [backend, caller.linked, caller.username, reads])

    return read && liveSeason(read.season, takesSince(read.before, takes))
}
