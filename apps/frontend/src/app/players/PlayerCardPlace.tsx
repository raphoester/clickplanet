import {PlayerInfoBackend} from "../../backends/player.ts"
import {PlayerReporter} from "../../backends/support.ts"
import PlayerCard from "./PlayerCard.tsx"
import {usePlayerCard} from "./playerCardStore.ts"

export type PlayerCardPlaceProps = {
    backend: PlayerInfoBackend
    accountId?: string
    reporter: PlayerReporter
}

export default function PlayerCardPlace({backend, accountId, reporter}: PlayerCardPlaceProps) {
    const player = usePlayerCard((card) => card.player)
    const close = usePlayerCard((card) => card.close)

    if (!player) return null
    return <PlayerCard key={player.accountId}
                       player={player}
                       backend={backend}
                       reporter={player.accountId === accountId ? undefined : reporter}
                       onClose={close}/>
}
