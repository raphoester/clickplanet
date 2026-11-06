import {useEffect} from "react"
import {BadgesBackend} from "../../backends/badges.ts"
import {useBadges} from "./badgesStore.ts"

// Reads the badges the page shows from the backend, and forgets them once it stops.
export function useBadgesFeed(backend: BadgesBackend): void {
    useEffect(() => {
        const {serve} = useBadges.getState()
        serve(backend)
        return () => serve(undefined)
    }, [backend])
}
