// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from 'vitest'
import {cleanup, fireEvent, render, screen} from '@testing-library/react'
import {useRef} from 'react'
import Bubble from './Bubble.tsx'

afterEach(cleanup)

function Anchored({box, onLost}: {box: Partial<DOMRect>, onLost?: () => void}) {
    const anchor = useRef<HTMLButtonElement | null>(null)
    return <>
        <button ref={(element) => {
            anchor.current = element
            if (element) element.getBoundingClientRect = () => ({left: 0, top: 0, width: 0, height: 0, right: 0, bottom: 0, ...box}) as DOMRect
        }}>Slot</button>
        <Bubble anchor={anchor} onLost={onLost}>Tap one of your tiles</Bubble>
    </>
}

describe("Bubble", () => {
    it("says its line over the element it points at", () => {
        render(<Anchored box={{left: 200, width: 40, top: 500, bottom: 540}}/>)

        const bubble = screen.getByRole("status")
        expect(bubble.textContent).toBe("Tap one of your tiles")
        expect(bubble.className).toContain("bubble--above")
        expect(bubble.style.left).toBe("220px")
        expect(bubble.style.top).toBe("490px")
    })

    it("goes under an element at the top of the screen", () => {
        render(<Anchored box={{left: 200, width: 40, top: 10, bottom: 50}}/>)

        const bubble = screen.getByRole("status")
        expect(bubble.className).not.toContain("bubble--above")
        expect(bubble.style.top).toBe("60px")
    })

    it("stays on the screen and still points at an element at its edge", () => {
        render(<Anchored box={{left: window.innerWidth - 30, width: 20, top: 500, bottom: 540}}/>)

        const bubble = screen.getByRole("status")
        expect(Number.parseFloat(bubble.style.left)).toBe(window.innerWidth - 128)
        expect(bubble.style.getPropertyValue("--bubble-arrow")).toBe("108px")
    })

    it("is drawn on the body, out of whatever clips its element", () => {
        const {container} = render(<Anchored box={{left: 200, width: 40, top: 500, bottom: 540}}/>)

        expect(container.querySelector(".bubble")).toBeNull()
        expect(document.body.querySelector(".bubble")).toBeTruthy()
    })

    it("says when the page moved under it", () => {
        const onLost = vi.fn()
        render(<Anchored box={{left: 200, width: 40, top: 500, bottom: 540}} onLost={onLost}/>)

        fireEvent.scroll(window)
        fireEvent(window, new Event("resize"))

        expect(onLost).toHaveBeenCalledTimes(2)
    })
})
