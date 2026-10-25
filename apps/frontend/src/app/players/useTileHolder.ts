import {useCallback, useEffect, useState} from "react"
import {TileHolders} from "../../backends/backend.ts"
import {PlayerLine, PlayerLinesBackend} from "../../backends/player.ts"
import {holderShown, Peek, Point} from "../../domain/tilePeek.ts"

export const LEAVE_GRACE_MS = 300

export type HolderShown = {
    line: PlayerLine
    takenAt: number
    point: Point
    touch: boolean
    id: number
}

export function useTileHolder(
    holders: TileHolders | undefined,
    players: PlayerLinesBackend | undefined,
    pointing: {peek: Peek, id: number} | undefined,
    ownAccountId: string | undefined,
): {holder: HolderShown | undefined, hold: (held: boolean) => void, dismiss: () => void} {
    const [holder, setHolder] = useState<HolderShown>()
    const [heldId, setHeldId] = useState<number>()

    useEffect(() => {
        if (!pointing || !holders || !players) return

        const {peek, id} = pointing
        const asking = new AbortController()
        const ask = async () => {
            const held = holderShown(await holders.tileHolder(peek.tile, asking.signal), peek, ownAccountId)
            const line = held && await players.playerLine(held.accountId, held.countryCode, asking.signal)
            if (asking.signal.aborted) return
            setHolder(held && line ? {line, takenAt: held.takenAt, point: peek.point, touch: peek.touch, id} : undefined)
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who holds the tile", e)
        })
        return () => asking.abort()
    }, [holders, players, pointing, ownAccountId])

    const held = holder !== undefined && heldId === holder.id
    useEffect(() => {
        if (pointing || held || !holder) return
        const timer = setTimeout(() => setHolder(undefined), LEAVE_GRACE_MS)
        return () => clearTimeout(timer)
    }, [pointing, held, holder])

    const id = holder?.id
    const hold = useCallback((on: boolean) => setHeldId(on ? id : undefined), [id])
    const dismiss = useCallback(() => setHolder(undefined), [])

    return {holder, hold, dismiss}
}
