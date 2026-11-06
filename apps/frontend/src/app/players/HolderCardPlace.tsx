import {useEffect} from "react"
import {TileHolder, TileHolders} from "../../backends/backend.ts"
import {PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import {DayMissions} from "../../backends/missions.ts"
import {CountryTiles} from "../../backends/progression.ts"
import {holderShown, Peek} from "../../domain/tilePeek.ts"
import HolderCard from "./HolderCard.tsx"
import {deletedLineOf, LEAVE_GRACE_MS, playerLineOf} from "./tileHolder.ts"
import {holderHeld, useTileHolder} from "./tileHolderStore.ts"

export type HolderCardPlaceProps = {
    holders: TileHolders
    players: PlayerInfoBackend
    pointing?: {peek: Peek, id: number}
    accountId?: string
    missions?: DayMissions
}

// Asks who holds the tile pointed at, keeps the answer in the store, and shows it.
export default function HolderCardPlace({holders, players, pointing, accountId, missions}: HolderCardPlaceProps) {
    const holder = useTileHolder((card) => card.holder)
    const held = useTileHolder(holderHeld)
    const show = useTileHolder((card) => card.show)
    const hold = useTileHolder((card) => card.hold)
    const dismiss = useTileHolder((card) => card.dismiss)

    useEffect(() => {
        if (!pointing) return

        const {peek, id} = pointing
        const asking = new AbortController()
        const ask = async () => {
            const held = holderShown(await holders.tileHolder(peek.tile, asking.signal), peek)
            const player = held && await playerOf(held, players)
            if (asking.signal.aborted) return
            show(held && player
                ? {
                    ...player, mine: held.accountId === accountId,
                    tile: {owner: peek.countryCode, ground: peek.ground, shields: peek.shields},
                    takenAt: held.takenAt, point: peek.point, touch: peek.touch, id,
                }
                : undefined)
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who holds the tile", e)
        })
        return () => asking.abort()
    }, [holders, players, pointing, accountId, show])

    useEffect(() => {
        if (pointing || held || !holder) return
        const timer = setTimeout(dismiss, LEAVE_GRACE_MS)
        return () => clearTimeout(timer)
    }, [pointing, held, holder, dismiss])

    // Nothing is left shown once the place goes.
    useEffect(() => dismiss, [dismiss])

    if (!holder) return null
    return <HolderCard key={`holder-${holder.id}`}
                       holder={holder}
                       missions={missions}
                       onHold={hold}
                       onClose={dismiss}/>
}

async function playerOf(held: TileHolder, players: PlayerInfoBackend): Promise<{line: PlayerLine, playsFor: CountryTiles[]} | undefined> {
    if (!held.accountId) return {line: deletedLineOf(held.countryCode), playsFor: []}
    const info = await players.playerInfo(held.accountId)
    return info && {line: playerLineOf(held.accountId, held.countryCode, info), playsFor: info.playsFor}
}
