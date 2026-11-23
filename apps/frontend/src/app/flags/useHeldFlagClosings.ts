import {useEffect, useRef} from "react"
import {AllegianceSource} from "../../backends/backend.ts"
import {useFlags} from "./flagsStore.ts"

// The server says whether the held flag is closed, since a lock of the rules keeps it playable: a close or a comeback
// the flags' stream tells reads the flag held again.
export function useHeldFlagClosings(source: AllegianceSource, held: string | undefined): void {
    const closed = useFlags((store) => held !== undefined && store.catalog.closedFlag(held))
    const seen = useRef<{held: string | undefined, closed: boolean}>()
    useEffect(() => {
        const was = seen.current
        seen.current = {held, closed}
        if (held === undefined || was?.held !== held || was.closed === closed) return
        source.refreshAllegiance().catch((e: unknown) => console.error("could not read the flag held again", e))
    }, [source, held, closed])
}
