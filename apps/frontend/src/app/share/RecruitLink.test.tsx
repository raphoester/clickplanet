// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Countries} from "../../domain/countries.ts"
import {ShareStats} from "../../domain/shareCard.ts"
import RecruitLink from "./RecruitLink.tsx"
import type {LinkDelivery} from "./deliverShare.ts"

vi.mock("./deliverShare.ts", async (original) => ({
    ...await original<typeof import("./deliverShare.ts")>(),
    linkDeliveryOffered: vi.fn(),
    deliverLink: vi.fn(),
}))

const {linkDeliveryOffered, deliverLink} = await import("./deliverShare.ts")
const offered = vi.mocked(linkDeliveryOffered)
const deliver = vi.mocked(deliverLink)

const FRANCE: ShareStats = {country: Countries.get("fr")!, rank: 2, tiles: 1500}
const JAPAN: ShareStats = {country: Countries.get("jp")!, rank: 1, tiles: 9000}
const button = (name: string) => screen.getByRole("button", {name}) as HTMLButtonElement

function setup(delivery: LinkDelivery = "copy") {
    offered.mockReturnValue(delivery)
    const view = render(<RecruitLink stats={FRANCE}/>)
    return {...view, user: userEvent.setup()}
}

beforeEach(() => {
    offered.mockReset()
    deliver.mockReset()
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("RecruitLink", () => {
    it("names the country played and shows its link", () => {
        setup()

        expect(screen.getByText("Recruit for France")).toBeDefined()
        expect(screen.getByText("clickplanet.lol/?f=fr")).toBeDefined()
    })

    it("follows the country when the player changes it", () => {
        const {rerender} = setup()

        rerender(<RecruitLink stats={JAPAN}/>)

        expect(screen.getByText("Recruit for Japan")).toBeDefined()
        expect(screen.getByText("clickplanet.lol/?f=jp")).toBeDefined()
    })

    it("copies the link, and says so on the button", async () => {
        const {user} = setup("copy")
        deliver.mockResolvedValue("copied")

        await user.click(button("Copy link"))

        expect(deliver).toHaveBeenCalledWith("copy", "https://clickplanet.lol/?f=fr&s=rc",
            "France is #2 on ClickPlanet with 1,500 tiles. Come and take them.")
        await waitFor(() => expect(button("Copied!")).toBeDefined())
    })

    it("gives a phone its share sheet", async () => {
        const {user} = setup("sheet")
        deliver.mockResolvedValue("shared")

        await user.click(button("Share link"))

        expect(deliver.mock.calls[0][0]).toBe("sheet")
        await waitFor(() => expect(button("Shared!")).toBeDefined())
    })

    it("goes quietly back to offering when the player closes the share sheet", async () => {
        const {user} = setup("sheet")
        deliver.mockResolvedValue("cancelled")

        await user.click(button("Share link"))

        await waitFor(() => expect(button("Share link").disabled).toBe(false))
    })

    it("refuses a second press while the first is going", async () => {
        const {user} = setup("sheet")
        let finish: (outcome: "shared") => void = () => {}
        deliver.mockReturnValue(new Promise((resolve) => {finish = resolve}))

        await user.click(button("Share link"))

        expect(button("One sec…").disabled).toBe(true)

        finish("shared")
        await waitFor(() => expect(button("Shared!")).toBeDefined())
    })

    it("offers another go when the link would not go, and logs why", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {user} = setup("copy")
        deliver.mockImplementation(() => Promise.reject(new Error("the clipboard said no")))

        await user.click(button("Copy link"))

        await waitFor(() => expect(button("Try again")).toBeDefined())
        expect(logged).toHaveBeenCalled()
    })

    it("goes back to offering once the outcome has been read", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const {user} = setup("copy")
        deliver.mockResolvedValue("copied")

        await user.click(button("Copy link"))
        await waitFor(() => expect(button("Copied!")).toBeDefined())

        await vi.advanceTimersByTimeAsync(3_000)
        expect(button("Copy link")).toBeDefined()

        vi.useRealTimers()
    })
})
