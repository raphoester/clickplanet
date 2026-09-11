// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ShareActions from "./ShareActions.tsx"

// What this browser offers and what each delivery does are deliverShare's, and
// tested there. What is under test here is only what the player sees happen.
vi.mock("./deliverShare.ts", async (original) => ({
    ...await original<typeof import("./deliverShare.ts")>(),
    deliveriesOffered: vi.fn(),
    deliverShare: vi.fn(),
}))

const {deliveriesOffered, deliverShare} = await import("./deliverShare.ts")
const offered = vi.mocked(deliveriesOffered)
const deliver = vi.mocked(deliverShare)

const file = new File([new Uint8Array([1])], "clickplanet-fr.png", {type: "image/png"})
const TEXT = "France is #2 on ClickPlanet. https://clickplanet.lol/?c=fr"
const button = (name: string) => screen.getByRole("button", {name})

function setup(deliveries: ReturnType<typeof deliveriesOffered> = ["copy", "download"]) {
    offered.mockReturnValue(deliveries)
    render(<ShareActions file={file} text={TEXT}/>)
    return userEvent.setup()
}

// Braces, not a one-liner: `mockReset` hands the mock back, and a `beforeEach`
// that returns something has vitest run it as the teardown.
beforeEach(() => {
    offered.mockReset()
    deliver.mockReset()
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
        deliver.mockResolvedValue("downloaded")

        await user.click(button("Save"))

        expect(deliver).toHaveBeenCalledWith("download", file, TEXT)
    })

    it("says which of the two actually happened, on the button it happened to", async () => {
        const user = setup(["copy", "download"])
        deliver.mockResolvedValue("copied")

        await user.click(button("Copy"))

        await waitFor(() => expect(button("Copied!")).toBeTruthy())
        expect(button("Save")).toBeTruthy()
    })

    it("says it is working, and refuses both buttons while it is", async () => {
        const user = setup(["copy", "download"])
        let finish: (outcome: "copied") => void = () => {}
        deliver.mockReturnValue(new Promise((resolve) => {finish = resolve}))

        await user.click(button("Copy"))

        expect(button("One sec…")).toBeTruthy()
        expect((button("Save") as HTMLButtonElement).disabled).toBe(true)

        finish("copied")
        await waitFor(() => expect(button("Copied!")).toBeTruthy())
    })

    it("goes quietly back to offering when the player closes the share sheet", async () => {
        const user = setup(["sheet"])
        deliver.mockResolvedValue("cancelled")

        await user.click(button("Share"))

        await waitFor(() => expect((button("Share") as HTMLButtonElement).disabled).toBe(false))
    })

    // A clipboard the browser refused reports the refusal rather than turning
    // into a download: the download is the button beside it.
    it("offers another go when a delivery would not go through, and logs why", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const user = setup(["copy", "download"])
        deliver.mockImplementation(() => Promise.reject(new Error("the clipboard said no")))

        await user.click(button("Copy"))

        await waitFor(() => expect(button("Try again")).toBeTruthy())
        expect(button("Save")).toBeTruthy()
        expect(logged).toHaveBeenCalled()
        logged.mockRestore()
    })

    it("goes back to offering once the outcome has been read", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const user = setup(["copy", "download"])
        deliver.mockResolvedValue("copied")

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
