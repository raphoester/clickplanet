import {useEffect, useRef} from "react"
import {ProgressionEvents, ProgressionListener} from "../../backends/progression.ts"
import {useHeldIdentity} from "../useHeldIdentity.ts"

// The titles and levels a signed-in account earns, live. A guest earns neither.
export function useProgressionEvents(backend: ProgressionListener, signedIn: boolean, on: ProgressionEvents): void {
    const token = useHeldIdentity(backend)
    const handlers = useRef(on)
    handlers.current = on

    useEffect(() => {
        if (!signedIn || !token) return
        return backend.listenForProgression({
            titleEarned: (title) => handlers.current.titleEarned(title),
            levelReached: (level) => handlers.current.levelReached(level),
        })
    }, [backend, signedIn, token])
}
