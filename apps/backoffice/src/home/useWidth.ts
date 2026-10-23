import {RefObject, useLayoutEffect, useRef, useState} from "react"

const FALLBACK = 640

// The width an element is laid out at, kept as it resizes. jsdom lays nothing out, so a test reads the fallback.
export function useWidth<T extends HTMLElement>(): [RefObject<T>, number] {
    const ref = useRef<T>(null)
    const [width, setWidth] = useState(FALLBACK)

    useLayoutEffect(() => {
        const element = ref.current
        if (element === null) return
        if (element.clientWidth > 0) setWidth(element.clientWidth)
        if (typeof ResizeObserver === "undefined") return

        const observer = new ResizeObserver(([entry]) => {
            if (entry.contentRect.width > 0) setWidth(Math.round(entry.contentRect.width))
        })
        observer.observe(element)
        return () => observer.disconnect()
    }, [])

    return [ref, width]
}
