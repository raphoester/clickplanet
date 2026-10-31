// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AllegianceLockedError} from "../../backends/backend.ts"
import {Allegiance} from "../../domain/allegiance.ts"
import {Countries} from "../../domain/countries.ts"
import InvitedModal from "./InvitedModal.tsx"
import {LOCKED_RULE} from "../CountryPicker.tsx"

afterEach(cleanup)

const BHUTAN = Countries.get("bt")!

function invited(held: Allegiance, onSwitch: () => Promise<void> = vi.fn(async () => undefined)) {
    const onClose = vi.fn()
    render(<InvitedModal friend={BHUTAN} held={held} onSwitch={onSwitch} onClose={onClose}/>)
    return {onSwitch, onClose}
}

describe("InvitedModal", () => {
    it("names the friend's flag, and switches to it", async () => {
        const {onSwitch, onClose} = invited({countryId: "bg"})
        expect(screen.getByText("Your friend plays for Bhutan.")).toBeDefined()

        await userEvent.click(screen.getByRole("button", {name: "Play for Bhutan"}))

        expect(onSwitch).toHaveBeenCalled()
        await waitFor(() => expect(onClose).toHaveBeenCalled())
    })

    it("keeps the flag held", async () => {
        const {onSwitch, onClose} = invited({countryId: "bg"})

        await userEvent.click(screen.getByRole("button", {name: "Keep Bulgaria"}))

        expect(onClose).toHaveBeenCalled()
        expect(onSwitch).not.toHaveBeenCalled()
    })

    it("says when a locked flag may change, and offers no switch until then", () => {
        invited({countryId: "bg", changeableAt: Date.now() + 30 * 60_000})

        expect(screen.getByText(/^You can switch at /)).toBeDefined()
        expect((screen.getByRole("button", {name: "Play for Bhutan"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("offers no switch while a battle's rules lock the flags", () => {
        invited({countryId: "bg", locked: true})

        expect(screen.getByText(LOCKED_RULE)).toBeDefined()
        expect((screen.getByRole("button", {name: "Play for Bhutan"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("closes on a flag the server says is locked, and stays open on another failure", async () => {
        const locked = invited({countryId: "bg"}, async () => {
            throw new AllegianceLockedError({countryId: "bg", changeableAt: Date.now() + 60_000})
        })
        await userEvent.click(screen.getByRole("button", {name: "Play for Bhutan"}))
        await waitFor(() => expect(locked.onClose).toHaveBeenCalled())
        cleanup()

        const errors = vi.spyOn(console, "error").mockImplementation(() => undefined)
        const failed = invited({countryId: "bg"}, async () => {
            throw new Error("network")
        })
        await userEvent.click(screen.getByRole("button", {name: "Play for Bhutan"}))
        expect(await screen.findByRole("alert")).toBeDefined()
        expect(failed.onClose).not.toHaveBeenCalled()
        errors.mockRestore()
    })
})
