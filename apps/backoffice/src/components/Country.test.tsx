// @vitest-environment jsdom
import {act, cleanup, render} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {Country} from "./Country.tsx"
import {learnFlags, useListedFlags} from "../flags.ts"

afterEach(() => {
    cleanup()
    learnFlags([])
})

// As the console does: it draws again once the list is read.
function Page({id}: {id: string}) {
    useListedFlags()
    return <Country id={id}/>
}

describe("Country", () => {
    it("draws a country of the game with its emoji and its name", () => {
        const {container} = render(<Country id="dz"/>)
        expect(container.textContent).toBe("\u{1F1E9}\u{1F1FF} Algeria")
        expect(container.querySelector(".flag-blank")).toBeNull()
    })

    it("draws a code it does not know with a plain grey flag, then the list's name once it is read", () => {
        const {container} = render(<Page id="r-k3p9x2ma0b"/>)
        expect(container.querySelector(".flag-blank")).not.toBeNull()
        expect(container.textContent).toBe(" R-K3P9X2MA0B")

        act(() => learnFlags([{code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}]))

        expect(container.textContent).toBe(" Bavaria")
        expect(container.querySelector(".flag-blank")).not.toBeNull()
    })

    it("draws a sold flag with its picture and its name once the list is read", () => {
        const {container} = render(<Page id="r-k3p9x2ma0b"/>)

        act(() => learnFlags([{
            code: "r-k3p9x2ma0b", name: "Bavaria", closed: false, picture: "https://media.test/flags/pictures/bavaria.png",
        }]))

        expect(container.textContent).toBe(" Bavaria")
        expect(container.querySelector("img.flag-picture")?.getAttribute("src")).toBe("https://media.test/flags/pictures/bavaria.png")
        expect(container.querySelector(".flag-blank")).toBeNull()
    })

    it("draws Brittany with no flag at all, as Unicode has none", () => {
        const {container} = render(<Country id="xb"/>)
        expect(container.querySelector(".flag")).toBeNull()
        expect(container.textContent).toBe("Brittany")
    })
})
