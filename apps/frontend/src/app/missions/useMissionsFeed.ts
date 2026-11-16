import {useEffect, useRef, useState} from "react"
import {MissionsBackend} from "../../backends/missions.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useReadsAfterClicks} from "../clicks/useAcceptedClicks.ts"
import {NO_MISSIONS, useMissions} from "./missionsStore.ts"

export function useMissionsFeed(
    backend: MissionsBackend,
    account: string | undefined,
    pledged: string | undefined,
    listenForClicks: ListenForClicks | undefined,
): void {
    const [read, setRead] = useState(0)
    const streamed = useRef(0)
    const reads = useReadsAfterClicks(listenForClicks)
    const endsAt = useMissions((missions) => missions.state.kind === "ready" ? missions.state.day.endsAt : undefined)

    useEffect(() => {
        let stale = false
        // A day the stream sent during the read is newer: the read could put a paid mission back to unpaid.
        const before = streamed.current
        backend.missions().then(
            (day) => {
                if (!stale && streamed.current === before) useMissions.getState().hear(day)
            },
            (e) => {
                console.error("Could not read the missions", e)
                if (!stale) useMissions.getState().fail()
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
            useMissions.getState().hear(day)
        })
    }, [backend, account])

    useEffect(() => {
        if (endsAt === undefined) return
        const turn = setTimeout(() => setRead((n) => n + 1), Math.max(0, endsAt - Date.now()) + 1000)
        return () => clearTimeout(turn)
    }, [endsAt])

    useEffect(() => () => useMissions.setState(NO_MISSIONS), [])
}
