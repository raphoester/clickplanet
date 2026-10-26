import {useCallback, useEffect, useState} from "react"
import {TileHolder, TileHolders} from "../../backends/backend.ts"
import {CountryTiles, NameColor, PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import {holderShown, Peek, Point} from "../../domain/tilePeek.ts"

export const LEAVE_GRACE_MS = 300

export const DELETED_ACCOUNT = "Deleted account"

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
            const player = held && await playerOf(held, players)
            if (asking.signal.aborted) return
            setHolder(held && player
                ? {
                    ...player, mine: held.accountId === ownAccountId,
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

async function playerOf(held: TileHolder, players: PlayerInfoBackend): Promise<{line: PlayerLine, playsFor: CountryTiles[]} | undefined> {
    if (!held.accountId) return {line: deletedLineOf(held.countryCode), playsFor: []}
    const info = await players.playerInfo(held.accountId)
    return info && {line: playerLineOf(held.accountId, held.countryCode, info), playsFor: info.playsFor}
}

export function deletedLineOf(countryCode: string): PlayerLine {
    return {
        accountId: "",
        name: DELETED_ACCOUNT,
        countryCode,
        guest: false,
        admin: false,
        color: NameColor.UNSPECIFIED,
        streak: 0,
    }
}

export function playerLineOf(accountId: string, countryCode: string, info: PlayerInfo): PlayerLine {
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
