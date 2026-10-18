import {useCallback, useEffect, useState} from "react"
import {pageOf, PageId} from "./navigation.ts"

// The page shown, kept in the address bar (#/reports): a reload or a link opens it again, and Back goes to the page
// before. Each page picked is a history entry.
export function usePage(): [PageId, (page: PageId) => void] {
    const [page, setPage] = useState(() => pageOf(window.location.hash))

    useEffect(() => {
        const onHash = () => setPage(pageOf(window.location.hash))
        window.addEventListener("hashchange", onHash)
        return () => window.removeEventListener("hashchange", onHash)
    }, [])

    const go = useCallback((to: PageId) => {
        if (window.location.hash !== hashOf(to)) window.location.hash = hashOf(to)
        setPage(to)
    }, [])

    return [page, go]
}

function hashOf(page: PageId): string {
    return `#/${page}`
}
