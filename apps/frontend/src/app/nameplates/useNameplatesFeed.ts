import {useEffect} from "react"
import {NameplatesBackend} from "../../backends/nameplates.ts"
import {useNameplates} from "./nameplatesStore.ts"

// Reads the nameplates the page shows from the backend, and forgets them once it stops.
export function useNameplatesFeed(backend: NameplatesBackend | undefined): void {
    useEffect(() => {
        if (!backend) return
        const {serve} = useNameplates.getState()
        serve(backend)
        return () => serve(undefined)
    }, [backend])
}
