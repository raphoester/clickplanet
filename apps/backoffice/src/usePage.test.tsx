// @vitest-environment jsdom
import {act, cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {pageOf} from "./navigation.ts"
import {usePage} from "./usePage.ts"

afterEach(() => {
    cleanup()
    window.location.hash = ""
})

function Shown() {
    const [page, go] = usePage()
    return (
        <>
            <p>{page}</p>
            <button type="button" onClick={() => go("replay")}>Replay</button>
        </>
    )
}

describe("pageOf", () => {
    it("reads the page an address names", () => {
        expect(pageOf("#/reports")).toBe("reports")
        expect(pageOf("#players")).toBe("players")
    })

    it("opens on the first page for an address that names none", () => {
        expect(pageOf("")).toBe("home")
        expect(pageOf("#/nowhere")).toBe("home")
    })
})

describe("usePage", () => {
    it("opens on the page in the address bar, as after a reload", () => {
        window.location.hash = "#/paint"
        render(<Shown/>)

        expect(screen.getByText("paint")).toBeTruthy()
    })

    it("puts the page picked in the address bar", () => {
        render(<Shown/>)

        fireEvent.click(screen.getByRole("button", {name: "Replay"}))

        expect(screen.getByText("replay")).toBeTruthy()
        expect(window.location.hash).toBe("#/replay")
    })

    it("follows the address bar, as Back does", async () => {
        render(<Shown/>)

        act(() => {
            window.location.hash = "#/titles"
        })

        await waitFor(() => expect(screen.getByText("titles")).toBeTruthy())
    })
})
