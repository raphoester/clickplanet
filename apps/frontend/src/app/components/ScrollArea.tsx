import {ReactNode, useEffect, useState} from "react";
import "./ScrollArea.css"

export type ScrollAreaProps = {
    className?: string;
    children: ReactNode;
}

export default function ScrollArea(props: ScrollAreaProps) {
    const [body, setBody] = useState<HTMLDivElement | null>(null)
    const more = useMoreBelow(body)

    return <div className={more ? "scroll-area scroll-area--more" : "scroll-area"}>
        <div ref={setBody} className={props.className ? `scroll-area-body ${props.className}` : "scroll-area-body"}>
            {props.children}
        </div>
    </div>
}

function useMoreBelow(scroller: HTMLElement | null): boolean {
    const [more, setMore] = useState(false)

    useEffect(() => {
        if (!scroller) return

        const measure = () => {
            const overflow = getComputedStyle(scroller).overflowY
            const scrolls = overflow === "auto" || overflow === "scroll"
            setMore(scrolls && scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight > 1)
        }
        const resizes = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(measure)
        const watch = () => {
            resizes?.disconnect()
            resizes?.observe(scroller)
            for (const child of scroller.children) resizes?.observe(child)
        }
        const mutations = new MutationObserver(() => {
            watch()
            measure()
        })

        watch()
        measure()
        mutations.observe(scroller, {childList: true})
        scroller.addEventListener("scroll", measure, {passive: true})

        return () => {
            scroller.removeEventListener("scroll", measure)
            mutations.disconnect()
            resizes?.disconnect()
        }
    }, [scroller])

    return more
}
