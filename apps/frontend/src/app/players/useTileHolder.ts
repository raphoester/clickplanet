import {useCallback, useEffect, useState} from "react"
import {TileHolder, TileHolders} from "../../backends/backend.ts"
import {CountryTiles, PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import {holderShown, Peek, Point} from "../../domain/tilePeek.ts"

export const LEAVE_GRACE_MS = 300

export type HolderShown = {
    line: PlayerLine
    playsFor: CountryTiles[]
    mine: boolean
    takenAt: number
    point: Point
    touch: boolean
    id: number
}

export function useTileHolder(
    holders: TileHolders | undefined,
    players: PlayerInfoBackend | undefined,
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
            const held = holderShown(await holders.tileHolder(peek.tile, asking.signal), peek)
            const info = held && await players.playerInfo(held.accountId)
            if (asking.signal.aborted) return
            setHolder(held && info
                ? {
                    line: lineOf(held, info), playsFor: info.playsFor, mine: held.accountId === ownAccountId,
                    takenAt: held.takenAt, point: peek.point, touch: peek.touch, id,
                }
                : undefined)
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

function lineOf({accountId, countryCode}: TileHolder, info: PlayerInfo): PlayerLine {
    return {
        accountId,
        name: info.name,
        countryCode,
        guest: info.guest,
        admin: info.admin,
        color: info.color,
        streak: info.streakCurrent,
        wornTitle: info.wornTitle,
    }
}
