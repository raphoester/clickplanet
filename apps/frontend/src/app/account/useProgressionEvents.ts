import {useEffect, useRef} from "react"
import {ProgressionEvents, ProgressionListener} from "../../backends/progression.ts"
import {useHeldIdentity} from "../useHeldIdentity.ts"

// The titles, levels and XP a signed-in account earns, live. A guest earns none.
export function useProgressionEvents(backend: ProgressionListener, signedIn: boolean, on: ProgressionEvents): void {
    const token = useHeldIdentity(backend)
    const handlers = useRef(on)
    handlers.current = on

    useEffect(() => {
        if (!signedIn || !token) return
        return backend.listenForProgression({
            titleEarned: (title) => handlers.current.titleEarned(title),
            levelReached: (level) => handlers.current.levelReached(level),
            xpGained: (gained) => handlers.current.xpGained(gained),
        })
    }, [backend, signedIn, token])
}
