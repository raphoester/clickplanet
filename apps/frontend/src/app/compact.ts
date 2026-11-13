import {useEffect, useState} from "react"

export const COMPACT = "(max-width: 768px), (max-height: 600px)"

// The desktop frame where the board and the chat start folded until the player opens one; see docs/screen.md.
export const SMALL_FRAME = "(max-width: 1599px)"

const matches = (query: string) => window.matchMedia?.(query).matches ?? false

function useMatches(query: string): boolean {
    const [matched, setMatched] = useState(() => matches(query))

    useEffect(() => {
        const list = window.matchMedia?.(query)
        if (!list?.addEventListener) return

        const change = () => setMatched(list.matches)
        list.addEventListener("change", change)
        return () => list.removeEventListener("change", change)
    }, [query])

    return matched
}

export const useCompact = () => useMatches(COMPACT)

export const useSmallFrame = () => useMatches(SMALL_FRAME)
