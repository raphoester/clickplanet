import {useEffect, useRef} from "react"
import {TileDeltas} from "../../domain/tileDeltas.ts"
import {useNews} from "../components/newsStore.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useOwnTakes} from "../viewer/useAcceptedClicks.ts"

// Once a page: the first time the flag the player took tiles for loses ground on the board.
export function useUnderAttack(
    flag: string,
    deltas: TileDeltas,
    listenForClicks: ListenForClicks | undefined,
) {
    const takes = useOwnTakes(listenForClicks)
    const playing = (takes.get(flag) ?? 0) > 0
    const shown = useRef(false)
    const tellAttack = useNews((news) => news.tellAttack)

    useEffect(() => {
        if (shown.current || !playing) return
        if ((deltas.get(flag)?.net ?? 0) >= 0) return
        shown.current = true
        tellAttack()
    }, [playing, deltas, flag, tellAttack])
}
