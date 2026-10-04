import {useEffect, useState} from "react"

export const COMPACT = "(max-width: 768px)"

export const opensFolded = () => window.matchMedia?.(COMPACT).matches ?? false

export function useCompact(): boolean {
    const [compact, setCompact] = useState(opensFolded)

    useEffect(() => {
        const query = window.matchMedia?.(COMPACT)
        if (!query?.addEventListener) return

        const change = () => setCompact(query.matches)
        query.addEventListener("change", change)
        return () => query.removeEventListener("change", change)
    }, [])

    return compact
}
