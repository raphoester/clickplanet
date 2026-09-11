// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ShareActions from "./ShareActions.tsx"
import {Countries} from "../../domain/countries.ts"
import {CapturedFrame} from "../viewer/capture.ts"

// The composition needs a real 2D canvas and the capture a real WebGL one, so
// what is under test here is only what the player sees the buttons do.
vi.mock("./shareGlobe.ts", () => ({shareGlobe: vi.fn()}))
vi.mock("./deliverShare.ts", async (original) => ({
    ...await original<typeof import("./deliverShare.ts")>(),
    deliveriesOffered: vi.fn(),
}))

const {shareGlobe} = await import("./shareGlobe.ts")
const {deliveriesOffered} = await import("./deliverShare.ts")
const share = vi.mocked(shareGlobe)
const offered = vi.mocked(deliveriesOffered)

const stats = {country: Countries.get("fr")!, rank: 2, tiles: 250}
const capture = () => Promise.resolve({} as CapturedFrame)
const button = (name: string) => screen.getByRole("button", {name})

function setup(deliveries: ReturnType<typeof deliveriesOffered> = ["copy", "download"]) {
    offered.mockReturnValue(deliveries)
    render(<ShareActions stats={stats} capture={capture}/>)
    return userEvent.setup()
}

// Braces, not a one-liner: `mockReset` hands the mock back, and a `beforeEach`
// that returns something has vitest run it as the teardown.
beforeEach(() => {
    share.mockReset()
    offered.mockReset()
})

afterEach(cleanup)

describe("ShareActions", () => {
    it("puts one button on screen per way this browser has of letting go", () => {
        setup(["copy", "download"])

        expect(button("Copy")).toBeTruthy()
        expect(button("Save")).toBeTruthy()
    })

    it("gives a phone its share sheet alone", () => {
        setup(["sheet"])

        expect(button("Share")).toBeTruthy()
        expect(screen.queryByRole("button", {name: "Copy"})).toBeNull()
    })

    it("asks for the delivery whose button was pressed, and no other", async () => {
        const user = setup(["copy", "download"])
        share.mockResolvedValue("downloaded")

        await user.click(button("Save"))

        expect(share).toHaveBeenCalledWith(capture, stats, "download")
    })

    it("says which of the two actually happened, on the button it happened to", async () => {
        const user = setup(["copy", "download"])
        share.mockResolvedValue("copied")

        await user.click(button("Copy"))

        await waitFor(() => expect(button("Copied!")).toBeTruthy())
        expect(button("Save")).toBeTruthy()
    })

    // There is one globe and one frame it is captured from, so a second press
    // while the first is drawing has nothing of its own to draw.
    it("says it is working, and refuses both buttons while it is", async () => {
        const user = setup(["copy", "download"])
        let finish: (outcome: "copied") => void = () => {}
        share.mockReturnValue(new Promise((resolve) => {finish = resolve}))

        await user.click(button("Copy"))

        expect(button("One sec…")).toBeTruthy()
        expect((button("Save") as HTMLButtonElement).disabled).toBe(true)

        finish("copied")
        await waitFor(() => expect(button("Copied!")).toBeTruthy())
    })

    it("goes quietly back to offering when the player closes the share sheet", async () => {
        const user = setup(["sheet"])
        share.mockResolvedValue("cancelled")

        await user.click(button("Share"))

        await waitFor(() => expect((button("Share") as HTMLButtonElement).disabled).toBe(false))
    })

    // A clipboard the browser refused reports the refusal rather than turning
    // into a download: the download is the button beside it.
    it("offers another go when a delivery would not go through, and logs why", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const user = setup(["copy", "download"])
        share.mockImplementation(() => Promise.reject(new Error("the clipboard said no")))

        await user.click(button("Copy"))

        await waitFor(() => expect(button("Try again")).toBeTruthy())
        expect(button("Save")).toBeTruthy()
        expect(logged).toHaveBeenCalled()
        logged.mockRestore()
    })

    it("goes back to offering once the outcome has been read", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const user = setup(["copy", "download"])
        share.mockResolvedValue("copied")

        await user.click(button("Copy"))
        await waitFor(() => expect(button("Copied!")).toBeTruthy())

        await vi.advanceTimersByTimeAsync(3_000)
        expect(button("Copy")).toBeTruthy()

        vi.useRealTimers()
    })

    it("announces the outcome rather than leaving it to be noticed", () => {
        setup(["download"])
        expect(screen.getByText("Save").getAttribute("aria-live")).toBe("polite")
    })
})
