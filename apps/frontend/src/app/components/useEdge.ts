import {useEffect} from "react";

// Publishes where the element ends on the screen, its bottom or its right edge, or its height, as a property of the
// page's root.
export function useEdge(element: HTMLElement | null, property: string, side: "bottom" | "right" | "height") {
    useEffect(() => {
        if (!element) return

        const style = document.documentElement.style
        const publish = () => {
            const box = element.getBoundingClientRect()
            if (box.width > 0 && box.height > 0) style.setProperty(property, `${Math.ceil(box[side])}px`)
            else style.removeProperty(property)
        }
        publish()

        const observer = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(publish)
        observer?.observe(element)

        return () => {
            observer?.disconnect()
            style.removeProperty(property)
        }
    }, [element, property, side])
}
