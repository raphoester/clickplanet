// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import App from "./App.tsx"
import type {Backends} from "./viewer/Viewer.tsx"
import {NO_FILTER} from "../domain/wordFilter.ts"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"

vi.mock("./viewer/Viewer.tsx", () => ({default: () => <div data-testid="viewer"/>}))

const backends = {
    backends: {} as Backends,
    words: NO_FILTER,
}

const modalShown = () => screen.queryByText("It's free and open source.") !== null

beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.setItem(COUNTRY_STORAGE_KEY, JSON.stringify({code: "fr", name: "France"}))
})
afterEach(() => {
    cleanup()
    localStorage.clear()
})

describe("App", () => {
    it("always renders the globe", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        render(<App {...backends}/>)
        expect(screen.getByTestId("viewer")).toBeDefined()
    })

    it("shows the donation modal on a high roll", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        render(<App {...backends}/>)
        expect(modalShown()).toBe(true)
    })

    it("hides it on a low roll", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.1)
        render(<App {...backends}/>)
        expect(modalShown()).toBe(false)
    })

    it("decides once, and does not re-roll on re-render", () => {
        const random = vi.spyOn(Math, "random").mockReturnValue(0.9)

        const {rerender} = render(<App {...backends}/>)
        expect(modalShown()).toBe(true)

        random.mockReturnValue(0.1)
        rerender(<App {...backends}/>)
        rerender(<App {...backends}/>)

        expect(modalShown()).toBe(true)
        expect(random).toHaveBeenCalledTimes(1)
    })
})
