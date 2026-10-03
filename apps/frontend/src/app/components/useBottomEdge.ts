import {useEffect} from "react";

export function useBottomEdge(element: HTMLElement | null, property: string) {
    useEffect(() => {
        if (!element) return

        const style = document.documentElement.style
        const publish = () => {
            const box = element.getBoundingClientRect()
            if (box.height > 0) style.setProperty(property, `${Math.ceil(box.bottom)}px`)
            else style.removeProperty(property)
        }
        publish()

        const observer = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(publish)
        observer?.observe(element)

        return () => {
            observer?.disconnect()
            style.removeProperty(property)
        }
    }, [element, property])
}
