import {CSSProperties, ReactNode, useEffect, useLayoutEffect, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {cardPlacement} from "../../domain/cardPlacement.ts"
import {Point} from "../../domain/tilePeek.ts"
import {CloseIcon} from "./icons.tsx"
import {useEscape} from "./useDialog.ts"
import "./PointerCard.css"

const CLICK_AFTER_TAP_MS = 600

export type PointerCardProps = {
    point: Point
    touch: boolean
    label: string
    className: string
    style?: CSSProperties
    // Anything that can change the card's size: it is placed again when this changes.
    shape?: unknown
    onHold: (held: boolean) => void
    onClose: () => void
    children: ReactNode
}

export default function PointerCard({point, touch, label, className, style, shape, onHold, onClose, children}: PointerCardProps) {
    const card = useRef<HTMLElement>(null)
    const [place, setPlace] = useState<Point>()
    useLayoutEffect(() => {
        const box = card.current?.getBoundingClientRect()
        if (!box) return
        setPlace(cardPlacement(point, box, {width: window.innerWidth, height: window.innerHeight}, touch))
    }, [point, touch, shape])

    useEscape(onClose)

    useEffect(() => {
        const outside = (event: PointerEvent) => {
            if (card.current?.contains(event.target as Node)) return
            if (event.pointerType !== "mouse") swallowNextClick()
            onClose()
        }
        document.addEventListener("pointerdown", outside, true)
        return () => document.removeEventListener("pointerdown", outside, true)
    }, [onClose])

    const placed = {
        ...style,
        left: `${place?.x ?? 0}px`,
        top: `${place?.y ?? 0}px`,
        visibility: place ? undefined : "hidden",
    } as CSSProperties

    return createPortal(
        <section ref={card}
                 className={`pointer-card panel ${className}`}
                 style={placed}
                 aria-label={label}
                 onPointerEnter={() => onHold(true)}
                 onPointerLeave={() => onHold(false)}>
            <button type="button" className="pointer-card-close" aria-label="Close" onClick={onClose}>
                <CloseIcon size={14}/>
            </button>
            {children}
        </section>,
        document.body)
}

// A tap that closes the card is not also a tap on what is under it: on the globe it would take a tile.
function swallowNextClick() {
    const swallow = (event: MouseEvent) => {
        event.stopPropagation()
        event.preventDefault()
    }
    window.addEventListener("click", swallow, {capture: true, once: true})
    setTimeout(() => window.removeEventListener("click", swallow, {capture: true}), CLICK_AFTER_TAP_MS)
}
