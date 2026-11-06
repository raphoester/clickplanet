import {useEffect, useRef, useState} from "react"
import {DayMissions, MissionsBackend} from "../../backends/missions.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useReadsAfterClicks} from "../viewer/useAcceptedClicks.ts"

export type MissionsState =
    | {kind: "loading"}
    | {kind: "failed"}
    | {kind: "ready", day: DayMissions}

export function useMissions(
    backend: MissionsBackend,
    account: string | undefined,
    pledged: string | undefined,
    listenForClicks: ListenForClicks | undefined,
): MissionsState {
    const [state, setState] = useState<MissionsState>({kind: "loading"})
    const [read, setRead] = useState(0)
    const streamed = useRef(0)
    const reads = useReadsAfterClicks(listenForClicks)
    const endsAt = state.kind === "ready" ? state.day.endsAt : undefined

    useEffect(() => {
        let stale = false
        // A day the stream sent during the read is newer: the read could put a paid mission back to unpaid.
        const before = streamed.current
        backend.missions().then(
            (day) => {
                if (!stale && streamed.current === before) setState({kind: "ready", day})
            },
            (e) => {
                console.error("Could not read the missions", e)
                if (!stale) setState((current) => current.kind === "ready" ? current : {kind: "failed"})
            },
        )
        return () => {
            stale = true
        }
    }, [backend, account, pledged, read, reads])

    useEffect(() => {
        if (!account) return
        return backend.follow((day) => {
            streamed.current++
            setState({kind: "ready", day})
        })
    }, [backend, account])

    useEffect(() => {
        if (endsAt === undefined) return
        const turn = setTimeout(() => setRead((n) => n + 1), Math.max(0, endsAt - Date.now()) + 1000)
        return () => clearTimeout(turn)
    }, [endsAt])

    return state
}
