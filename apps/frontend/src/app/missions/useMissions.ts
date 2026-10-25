import {useCallback, useEffect, useState} from "react"
import {DayMissions, MissionsBackend} from "../../backends/missions.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useReadsAfterClicks} from "../viewer/useAcceptedClicks.ts"

export type MissionsState =
    | {kind: "loading"}
    | {kind: "failed"}
    | {kind: "ready", day: DayMissions}

export function useMissions(
    backend: MissionsBackend | undefined,
    countryCode: string,
    listenForClicks: ListenForClicks | undefined,
): [MissionsState, () => void] {
    const [state, setState] = useState<MissionsState>({kind: "loading"})
    const [read, setRead] = useState(0)
    const reads = useReadsAfterClicks(listenForClicks)
    const endsAt = state.kind === "ready" ? state.day.endsAt : undefined

    useEffect(() => {
        if (!backend) return
        let stale = false
        backend.missions(countryCode).then(
            (day) => {
                if (!stale) setState({kind: "ready", day})
            },
            (e) => {
                console.error("Could not read the missions", e)
                if (!stale) setState((current) => current.kind === "ready" ? current : {kind: "failed"})
            },
        )
        return () => {
            stale = true
        }
    }, [backend, countryCode, read, reads])

    useEffect(() => {
        if (endsAt === undefined) return
        const turn = setTimeout(() => setRead((n) => n + 1), Math.max(0, endsAt - Date.now()) + 1000)
        return () => clearTimeout(turn)
    }, [endsAt])

    const reread = useCallback(() => setRead((n) => n + 1), [])
    return [state, reread]
}
