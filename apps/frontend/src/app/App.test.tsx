// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import App from "./App.tsx"
import type {OwnershipsGetter, TileClicker, UpdatesListener} from "../backends/backend.ts"

vi.mock("./viewer/Viewer.tsx", () => ({default: () => <div data-testid="viewer"/>}))

const backends = {
    tileClicker: {} as TileClicker,
    ownershipsGetter: {} as OwnershipsGetter,
    updatesListener: {} as UpdatesListener,
}

const modalShown = () => screen.queryByText("Do you like ClickPlanet ?") !== null

beforeEach(() => vi.restoreAllMocks())
afterEach(cleanup)

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

    /**
     * The roll used to be a bare `Math.random()` in the returned JSX. React may
     * render a component more than once for one commit, so the modal could
     * appear or vanish on any re-render.
     */
    it("decides once, and does not re-roll on re-render", () => {
        const random = vi.spyOn(Math, "random").mockReturnValue(0.9)

        const {rerender} = render(<App {...backends}/>)
        expect(modalShown()).toBe(true)

        random.mockReturnValue(0.1) // a roll that would have hidden it
        rerender(<App {...backends}/>)
        rerender(<App {...backends}/>)

        expect(modalShown()).toBe(true)
        expect(random).toHaveBeenCalledTimes(1)
    })
})
