import {useEffect, useState} from "react"
import {PlayerInfo, PlayerInfoBackend, RosterEntry} from "../../backends/player.ts"

export type PlayerInfoState =
    | {kind: "loading"}
    /** A guest has no username, so there is nothing to ask for. */
    | {kind: "guest"}
    /** No player holds the name any more: renamed, or the account is gone. */
    | {kind: "missing"}
    | {kind: "failed"}
    | {kind: "ready", info: PlayerInfo}

/** What the server knows about `player`, read once when it is opened. A guest asks nothing. */
export function usePlayerInfo(backend: PlayerInfoBackend, player: RosterEntry): PlayerInfoState {
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
