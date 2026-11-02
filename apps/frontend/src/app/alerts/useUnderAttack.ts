import {useCallback, useEffect, useRef, useState} from "react"
import {TileDeltas} from "../../domain/tileDeltas.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useOwnTakes} from "../viewer/useAcceptedClicks.ts"

export type UnderAttack = {
    moment?: {at: number}
    dismiss: () => void
}

// Once a page: the first time the flag the player took tiles for loses ground on the board.
export function useUnderAttack(
    armed: boolean,
    flag: string,
    deltas: TileDeltas,
    listenForClicks: ListenForClicks | undefined,
): UnderAttack {
    const takes = useOwnTakes(listenForClicks)
    const playing = (takes.get(flag) ?? 0) > 0
    const shown = useRef(false)
    const [moment, setMoment] = useState<{at: number}>()

    useEffect(() => {
        if (shown.current || !armed || !playing) return
        if ((deltas.get(flag)?.net ?? 0) >= 0) return
        shown.current = true
        setMoment({at: Date.now()})
    }, [armed, playing, deltas, flag])

    const dismiss = useCallback(() => setMoment(undefined), [])

    return {moment, dismiss}
}
