import {useEffect, useState} from "react"

// Whether the dark scheme is on, kept as it changes: a canvas reads its colors when it draws, so it draws again.
export function useDarkScheme(): boolean {
    const [dark, setDark] = useState(() => schemeQuery()?.matches ?? false)

    useEffect(() => {
        const query = schemeQuery()
        if (query === undefined) return
        const follow = (event: MediaQueryListEvent) => setDark(event.matches)
        query.addEventListener("change", follow)
        return () => query.removeEventListener("change", follow)
    }, [])

    return dark
}

function schemeQuery(): MediaQueryList | undefined {
    return typeof window.matchMedia === "function" ? window.matchMedia("(prefers-color-scheme: dark)") : undefined
}
