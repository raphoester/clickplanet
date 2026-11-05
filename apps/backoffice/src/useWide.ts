import {useSyncExternalStore} from "react"

const WIDE = "(min-width: 1024px)"

// A wide screen keeps the menu beside the page; a narrower one opens it from the burger.
export function useWide(): boolean {
    return useSyncExternalStore(subscribe, () => window.matchMedia(WIDE).matches)
}

function subscribe(onChange: () => void) {
    const query = window.matchMedia(WIDE)
    query.addEventListener("change", onChange)
    return () => query.removeEventListener("change", onChange)
}
