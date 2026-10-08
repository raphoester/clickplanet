import {CSSProperties, ReactNode, RefObject, useLayoutEffect, useState} from 'react'
import {createPortal} from 'react-dom'
import './Bubble.css'

export const BUBBLE_MS = 3500

const MAX_WIDTH_PX = 240
const GAP_PX = 10
const EDGE_PX = 8
const ROOM_PX = 72

type Place = {
    left: number
    top: number
    arrow: number
    above: boolean
}

export type BubbleProps = {
    anchor: RefObject<HTMLElement | null>
    id?: string
    gap?: number
    onLost?: () => void
    children: ReactNode
}

export default function Bubble({anchor, id, gap = GAP_PX, onLost, children}: BubbleProps) {
    const [place, setPlace] = useState<Place | undefined>()

    useLayoutEffect(() => {
        setPlace(placeOver(anchor.current, gap))
    }, [anchor, gap, children])

    useLayoutEffect(() => {
        if (!onLost) return
        window.addEventListener("scroll", onLost, true)
        window.addEventListener("resize", onLost)
        return () => {
            window.removeEventListener("scroll", onLost, true)
            window.removeEventListener("resize", onLost)
        }
    }, [onLost])

    if (!place) return null

    // On the body: a panel that scrolls or clips would cut it.
    return createPortal(
        <div id={id}
             role="status"
             className={place.above ? "bubble bubble--above" : "bubble"}
             style={{left: `${place.left}px`, top: `${place.top}px`, maxWidth: `${MAX_WIDTH_PX}px`, "--bubble-arrow": `${place.arrow}px`} as CSSProperties}>
            {children}
        </div>,
        document.body)
}

function placeOver(anchor: HTMLElement | null, gap: number): Place | undefined {
    const box = anchor?.getBoundingClientRect()
    if (!box) return undefined

    const half = Math.min(MAX_WIDTH_PX, window.innerWidth - 2 * EDGE_PX) / 2
    const middle = box.left + box.width / 2
    const left = Math.min(Math.max(middle, half + EDGE_PX), window.innerWidth - half - EDGE_PX)
    const above = box.top > ROOM_PX

    return {
        left,
        top: above ? box.top - gap : box.bottom + gap,
        arrow: middle - left,
        above,
    }
}
