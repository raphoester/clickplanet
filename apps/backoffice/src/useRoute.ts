import {useCallback, useEffect, useState} from "react"
import {hashOf, Route, routeOf} from "./navigation.ts"

// The page shown and what is open on it, kept in the address bar (#/reports/report-3): a reload or a link opens it
// again, and Back goes to the route before. Each route picked is a history entry.
export function useRoute(): [Route, (route: Route) => void] {
    const [route, setRoute] = useState(() => routeOf(window.location.hash))

    useEffect(() => {
        const onHash = () => setRoute(routeOf(window.location.hash))
        window.addEventListener("hashchange", onHash)
        return () => window.removeEventListener("hashchange", onHash)
    }, [])

    const go = useCallback((to: Route) => {
        if (window.location.hash !== hashOf(to)) window.location.hash = hashOf(to)
        setRoute(to)
    }, [])

    return [route, go]
}
