// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ShareButton from "./ShareButton.tsx"
import {Countries} from "../../domain/countries.ts"
import {CapturedFrame} from "../viewer/capture.ts"

// The composition needs a real 2D canvas and the capture a real WebGL one, so
// what is under test here is only what the player sees the button do.
vi.mock("./shareGlobe.ts", () => ({shareGlobe: vi.fn()}))
const {shareGlobe} = await import("./shareGlobe.ts")
const share = vi.mocked(shareGlobe)

const stats = {country: Countries.get("fr")!, rank: 2, tiles: 250}
const capture = () => Promise.resolve({} as CapturedFrame)
const button = () => screen.getByRole("button")

function setup() {
    render(<ShareButton stats={stats} capture={capture}/>)
    return userEvent.setup()
}

// Braces, not a one-liner: `mockReset` hands the mock back, and a `beforeEach`
// that returns something has vitest run it as the teardown.
beforeEach(() => {
    share.mockReset()
})
afterEach(cleanup)

describe("ShareButton", () => {
    it("offers to share before it is pressed", () => {
        setup()
        expect(button().textContent).toBe("Share")
    })

    it("passes the standing on the card straight through", async () => {
        const user = setup()
        share.mockResolvedValue("copied")

        await user.click(button())

        expect(share).toHaveBeenCalledWith(capture, stats)
    })

    it("says which of the three ways the image actually went out", async () => {
        const user = setup()
        share.mockResolvedValue("downloaded")

        await user.click(button())

        await waitFor(() => expect(button().textContent).toBe("Saved!"))
    })

    it("says it is working, and refuses a second press while it is", async () => {
        const user = setup()
        let finish: (outcome: "shared") => void = () => {}
        share.mockReturnValue(new Promise((resolve) => {finish = resolve}))

        await user.click(button())

        expect(button().textContent).toBe("Drawing…")
        expect((button() as HTMLButtonElement).disabled).toBe(true)

        finish("shared")
        await waitFor(() => expect(button().textContent).toBe("Shared!"))
    })

    // The sheet closing is the player saying no, and a button that then reports
    // something is a button that did something they did not ask for.
    it("goes quietly back to offering when the player closes the share sheet", async () => {
        const user = setup()
        share.mockResolvedValue("cancelled")

        await user.click(button())

        await waitFor(() => expect((button() as HTMLButtonElement).disabled).toBe(false))
        expect(button().textContent).toBe("Share")
    })

    it("offers another go when it could not build the image, and logs why", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const user = setup()
        share.mockImplementation(() => Promise.reject(new Error("no WebGL context")))

        await user.click(button())

        await waitFor(() => expect(button().textContent).toBe("Try again"))
        expect((button() as HTMLButtonElement).disabled).toBe(false)
        expect(logged).toHaveBeenCalled()
        logged.mockRestore()
    })

    it("goes back to offering once the outcome has been read", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const user = setup()
        share.mockResolvedValue("copied")

        await user.click(button())
        await waitFor(() => expect(button().textContent).toBe("Copied!"))

        await vi.advanceTimersByTimeAsync(3_000)
        expect(button().textContent).toBe("Share")

        vi.useRealTimers()
    })

    it("announces the outcome rather than leaving it to be noticed", () => {
        setup()
        expect(screen.getByText("Share").getAttribute("aria-live")).toBe("polite")
    })
})
