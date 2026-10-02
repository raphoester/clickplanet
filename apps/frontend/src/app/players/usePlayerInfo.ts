import {useEffect, useState} from "react"
import {PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"

export type PlayerInfoState =
    | {kind: "loading"}
    | {kind: "guest"}
    | {kind: "missing"}
    | {kind: "failed"}
    | {kind: "ready", info: PlayerInfo}

export function usePlayerInfo(backend: PlayerInfoBackend, player: PlayerLine): PlayerInfoState {
    const [state, setState] = useState<PlayerInfoState>(() => player.guest ? {kind: "guest"} : {kind: "loading"})

    useEffect(() => {
        if (player.guest) {
            setState({kind: "guest"})
            return
        }

        let stale = false
        setState({kind: "loading"})
        backend.playerInfo(player.name).then(
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
    }, [backend, player.name, player.guest])

    return state
}
