import {PointerEvent, useCallback, useEffect, useRef, useState} from "react"
import {TeaserCard} from "../../backends/promo.ts"
import {ChevronIcon} from "../components/icons.tsx"

// A card a few seconds: long enough to read a headline and its line.
export const TEASER_TURN_MS = 5000

// How far a finger goes before a swipe turns the card.
const SWIPE_PX = 40

function reducedMotion(): boolean {
    return typeof window.matchMedia === "function" && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

// The teaser's cards in a row, one shown at a time: swiped on a phone, turned with the arrows or the dots (the
// keyboard's way in), and turning on their own until the player touches them. Under reduced motion they never turn on
// their own. The row is moved by a transform, so the card shown is always the one the dots say.
export default function TeaserCarousel({cards, turnMs = TEASER_TURN_MS}: {cards: readonly TeaserCard[], turnMs?: number}) {
    const [shown, setShown] = useState(0)
    const [touched, setTouched] = useState(false)
    const [drag, setDrag] = useState<number>()
    const swipe = useRef<{x: number, pointer: number}>()
    const [still] = useState(reducedMotion)
    const many = cards.length > 1

    const go = useCallback((index: number) => setShown((index + cards.length) % cards.length), [cards.length])

    useEffect(() => {
        if (!many || touched || still) return
        const timer = setTimeout(() => go(shown + 1), turnMs)
        return () => clearTimeout(timer)
    }, [go, many, shown, still, touched, turnMs])

    const step = (by: number) => {
        setTouched(true)
        go(shown + by)
    }

    const down = (event: PointerEvent) => {
        setTouched(true)
        if (!many) return
        swipe.current = {x: event.clientX, pointer: event.pointerId}
        event.currentTarget.setPointerCapture?.(event.pointerId)
    }
    const move = (event: PointerEvent) => {
        if (swipe.current?.pointer === event.pointerId) setDrag(event.clientX - swipe.current.x)
    }
    const up = (event: PointerEvent) => {
        if (swipe.current?.pointer !== event.pointerId) return
        const moved = event.clientX - swipe.current.x
        swipe.current = undefined
        setDrag(undefined)
        if (moved <= -SWIPE_PX) go(shown + 1)
        else if (moved >= SWIPE_PX) go(shown - 1)
    }
    const cancel = (event: PointerEvent) => {
        if (swipe.current?.pointer !== event.pointerId) return
        swipe.current = undefined
        setDrag(undefined)
    }

    return (
        <section className="teaser-carousel" aria-roledescription="carousel" aria-label="What the battle holds">
            <div className="teaser-track"
                 onPointerDown={down}
                 onPointerMove={move}
                 onPointerUp={up}
                 onPointerCancel={cancel}>
                <div className="teaser-row"
                     data-dragging={drag !== undefined}
                     style={{transform: `translateX(calc(${-shown * 100}% + ${drag ?? 0}px))`}}>
                    {cards.map((card, i) => (
                        <figure
                            key={i}
                            className="teaser-card"
                            aria-roledescription="slide"
                            aria-label={`${i + 1} of ${cards.length}`}
                            aria-hidden={i !== shown}
                        >
                            <img src={card.backgroundUrl} alt="" loading={i === 0 ? "eager" : "lazy"} draggable={false}/>
                            <figcaption>
                                <strong className="teaser-card-headline">{card.headline}</strong>
                                {card.line !== "" && <span className="teaser-card-line">{card.line}</span>}
                            </figcaption>
                        </figure>
                    ))}
                </div>
            </div>
            {many && (
                <>
                    <button type="button" className="teaser-arrow teaser-arrow--previous" aria-label="Previous card"
                            onClick={() => step(-1)}><ChevronIcon size={22}/></button>
                    <button type="button" className="teaser-arrow teaser-arrow--next" aria-label="Next card"
                            onClick={() => step(1)}><ChevronIcon size={22}/></button>
                    <div className="teaser-dots">
                        {cards.map((_, i) => (
                            <button
                                key={i}
                                type="button"
                                className="teaser-dot"
                                aria-label={`Card ${i + 1} of ${cards.length}`}
                                aria-current={i === shown}
                                onClick={() => {
                                    setTouched(true)
                                    go(i)
                                }}
                            />
                        ))}
                    </div>
                </>
            )}
        </section>
    )
}
