// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, fireEvent, render} from "@testing-library/react"
import ScrollArea from "./ScrollArea.tsx"

afterEach(cleanup)

function area(overflow: string, sizes: {content: number, box: number}) {
    const {container} = render(<ScrollArea className="body"><p>Privacy policy</p></ScrollArea>)
    const wrapper = container.querySelector(".scroll-area") as HTMLElement
    const body = container.querySelector(".body") as HTMLElement
    body.style.overflowY = overflow
    Object.defineProperty(body, "scrollHeight", {configurable: true, value: sizes.content})
    Object.defineProperty(body, "clientHeight", {configurable: true, value: sizes.box})
    fireEvent.scroll(body)
    return {wrapper, body}
}

function scrollTo(body: HTMLElement, top: number) {
    act(() => {
        body.scrollTop = top
        fireEvent.scroll(body)
    })
}

describe("ScrollArea", () => {
    it("draws no fade when everything fits", () => {
        const {wrapper} = area("auto", {content: 300, box: 300})

        expect(wrapper.classList.contains("scroll-area--more")).toBe(false)
    })

    it("draws the fade while more is below, and not at the end", () => {
        const {wrapper, body} = area("auto", {content: 500, box: 300})
        expect(wrapper.classList.contains("scroll-area--more")).toBe(true)

        scrollTo(body, 150)
        expect(wrapper.classList.contains("scroll-area--more")).toBe(true)

        scrollTo(body, 200)
        expect(wrapper.classList.contains("scroll-area--more")).toBe(false)
    })

    it("draws no fade on a body that does not scroll", () => {
        const {wrapper} = area("hidden", {content: 500, box: 300})

        expect(wrapper.classList.contains("scroll-area--more")).toBe(false)
    })
})
