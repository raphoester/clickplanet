import {useEffect, useState} from "react"
import {PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"

export type PlayerInfoState =
    | {kind: "loading"}
    | {kind: "guest"}
    | {kind: "missing"}
    | {kind: "failed"}
    | {kind: "ready", info: PlayerInfo}

export function usePlayerInfo(
    backend: PlayerInfoBackend,
    {accountId, guest}: Pick<PlayerLine, "accountId" | "guest">,
): PlayerInfoState {
    const [state, setState] = useState<PlayerInfoState>(() => unasked(accountId, guest) ?? {kind: "loading"})

    useEffect(() => {
        const known = unasked(accountId, guest)
        if (known) {
            setState(known)
            return
        }

        let stale = false
        setState({kind: "loading"})
        backend.playerInfo(accountId).then(
            (info) => {
                if (!stale) setState(info ? {kind: "ready", info} : {kind: "missing"})
            },
            (e) => {
                if (stale) return
                console.error("Could not read the player", e)
                setState({kind: "failed"})
            },
        )
        return () => {
            stale = true
        }
    }, [backend, accountId, guest])

    return state
}

function unasked(accountId: string, guest: boolean): PlayerInfoState | undefined {
    if (guest) return {kind: "guest"}
    if (accountId === "") return {kind: "missing"}
    return undefined
}
