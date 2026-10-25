import {useCallback, useEffect, useState} from "react"
import {TileHolders} from "../../backends/backend.ts"
import {PlayerLine, PlayerLinesBackend} from "../../backends/player.ts"
import {holderShown, Peek} from "../../domain/tilePeek.ts"

export type HolderMoment = {
    line: PlayerLine
    takenAt: number
    id: number
    at: number
}

export function useTileHolder(
    holders: TileHolders | undefined,
    players: PlayerLinesBackend | undefined,
    peeked: {peek: Peek, id: number} | undefined,
    ownAccountId: string | undefined,
): {holder: HolderMoment | undefined, dismiss: () => void} {
    const [holder, setHolder] = useState<HolderMoment>()

    useEffect(() => {
        if (!peeked || !holders || !players) return

        const asking = new AbortController()
        const ask = async () => {
            const held = holderShown(await holders.tileHolder(peeked.peek.tile, asking.signal), peeked.peek, ownAccountId)
            if (!held) return
            const line = await players.playerLine(held.accountId, held.countryCode, asking.signal)
            if (!line || asking.signal.aborted) return
            setHolder((was) => ({line, takenAt: held.takenAt, id: (was?.id ?? 0) + 1, at: performance.now()}))
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who holds the tile", e)
        })
        return () => asking.abort()
    }, [holders, players, peeked, ownAccountId])

    const dismiss = useCallback(() => setHolder(undefined), [])

    return {holder, dismiss}
}
