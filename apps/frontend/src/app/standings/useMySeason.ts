import {useEffect, useState} from "react"
import {NameColor} from "../../backends/player.ts"
import {MySeason, StandingsBackend} from "../../backends/standings.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"

export const AFTER_CLICKS_MS = 3_000

export type Caller = {
    username?: string
    color?: NameColor
    linked: boolean
}

export function useMySeason(backend: StandingsBackend, caller: Caller, listenForClicks: ListenForClicks): MySeason | undefined {
    const [season, setSeason] = useState<MySeason>()
    const [read, setRead] = useState(0)

    useEffect(() => {
        let timer: ReturnType<typeof setTimeout> | undefined
        const stop = listenForClicks(() => {
            clearTimeout(timer)
            timer = setTimeout(() => setRead((n) => n + 1), AFTER_CLICKS_MS)
        })
        return () => {
            stop()
            clearTimeout(timer)
        }
    }, [listenForClicks])

    useEffect(() => {
        let stale = false
        backend.mySeason().then(
            (mine) => {
                if (!stale) setSeason(mine)
            },
            (e) => console.error("Could not read your season", e),
        )
        return () => {
            stale = true
        }
    }, [backend, caller.linked, caller.username, read])

    return season
}
