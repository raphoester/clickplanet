import {useSyncExternalStore} from "react"

export function usePageVisible(): boolean {
    return useSyncExternalStore(subscribe, () => document.visibilityState === "visible")
}

function subscribe(onChange: () => void) {
    document.addEventListener("visibilitychange", onChange)
    return () => document.removeEventListener("visibilitychange", onChange)
}
