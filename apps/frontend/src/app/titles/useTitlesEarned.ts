import {useEffect, useRef} from "react"
import {PlayerTitle, TitlesListener} from "../../backends/progression.ts"
import {useHeldIdentity} from "../useHeldIdentity.ts"

export function useTitlesEarned(
    backend: TitlesListener,
    signedIn: boolean,
    onEarned: (title: PlayerTitle) => void,
): void {
    const token = useHeldIdentity(backend)
    const earned = useRef(onEarned)
    earned.current = onEarned

    useEffect(() => {
        if (!signedIn || !token) return
        return backend.listenForTitles((title) => earned.current(title))
    }, [backend, signedIn, token])
}
