// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import InviteFriends from "./InviteFriends.tsx"
import {Countries} from "../../domain/countries.ts"

vi.mock("./deliverShare.ts", async (original) => ({
    ...await original<typeof import("./deliverShare.ts")>(),
    copyLink: vi.fn(async () => "copied"),
}))

const {copyLink} = await import("./deliverShare.ts")

const FRANCE = Countries.get("fr")!

afterEach(() => {
    cleanup()
    vi.clearAllMocks()
})

describe("InviteFriends", () => {
    it("copies CrazyGames' link that invites to the flag played", async () => {
        const invite = vi.fn(async (flag: string) => `https://www.crazygames.com/game/clickplanet?czy_invite=true&f=${flag}`)
        render(<InviteFriends country={FRANCE} invite={invite}/>)

        expect(screen.getByText("Recruit for France")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Invite friends"}))

        expect(await screen.findByRole("button", {name: "Copied!"})).toBeTruthy()
        expect(invite).toHaveBeenCalledWith("fr")
        expect(copyLink).toHaveBeenCalledWith("https://www.crazygames.com/game/clickplanet?czy_invite=true&f=fr")
    })

    it("offers to try again when CrazyGames gives no link", async () => {
        const errors = vi.spyOn(console, "error").mockImplementation(() => undefined)
        render(<InviteFriends country={FRANCE} invite={async () => undefined}/>)

        fireEvent.click(screen.getByRole("button", {name: "Invite friends"}))

        expect(await screen.findByRole("button", {name: "Try again"})).toBeTruthy()
        expect(copyLink).not.toHaveBeenCalled()
        errors.mockRestore()
    })
})
