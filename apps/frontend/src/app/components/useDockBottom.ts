import {useEffect} from "react";

export const DOCK_BOTTOM = "--click-budget-dock-bottom"

export function useDockBottom(dock: HTMLElement | null) {
    useEffect(() => {
        if (!dock) return

        const style = document.documentElement.style
        const publish = () => style.setProperty(DOCK_BOTTOM, `${Math.ceil(dock.getBoundingClientRect().bottom)}px`)
        publish()

        const observer = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(publish)
        observer?.observe(dock)

        return () => {
            observer?.disconnect()
            style.removeProperty(DOCK_BOTTOM)
        }
    }, [dock])
}
