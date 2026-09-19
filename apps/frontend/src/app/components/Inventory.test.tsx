// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {cleanup, fireEvent, render, screen} from '@testing-library/react'
import Inventory, {INVENTORY_FOLDED_KEY, InventoryProps} from './Inventory.tsx'
import {ALL_OFF, NO_CHARGES} from "../../domain/bonus.ts"

afterEach(cleanup)
beforeEach(() => window.localStorage.clear())

const rules = {blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3}

function inventory(props: Partial<InventoryProps> = {}) {
    return <Inventory charges={NO_CHARGES}
                      rules={rules}
                      switches={ALL_OFF}
                      onToggle={() => {}}
                      bombArmed={false}
                      onToggleBomb={() => {}}
                      onUseRefill={() => true}
                      {...props}/>
}

const slot = (name: RegExp) => screen.getByRole("button", {name}) as HTMLButtonElement

describe("Inventory", () => {
    it("shows every slot, and none can be pressed while nothing is held", () => {
        render(inventory())

        for (const name of [/^Refill/, /^Bomb/, /^Spread/, /^Enclose/]) expect(slot(name).disabled).toBe(true)
    })

    it("counts the pools against how many can be held", () => {
        render(inventory({charges: {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2}}))

        expect(slot(/^Spread, 5\/8/)).toBeTruthy()
        expect(slot(/^Enclose, 2\/3/)).toBeTruthy()
    })

    it("switches spread and enclose, and says which are on", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2}
        const {rerender} = render(inventory({charges, onToggle: toggle}))

        expect(slot(/^Spread/).getAttribute("aria-pressed")).toBe("false")
        fireEvent.click(slot(/^Spread/))
        fireEvent.click(slot(/^Enclose/))
        expect(toggle.mock.calls).toEqual([["spread"], ["enclose"]])

        rerender(inventory({charges, onToggle: toggle, switches: {spread: true, enclose: false}}))
        expect(slot(/^Spread/).getAttribute("aria-pressed")).toBe("true")
        expect(slot(/^Spread, 5\/8, On/)).toBeTruthy()
        expect(slot(/^Enclose/).getAttribute("aria-pressed")).toBe("false")
    })

    it("aims the bomb and puts it away", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, bomb: true}
        const {rerender} = render(inventory({charges, onToggleBomb: toggle}))

        fireEvent.click(slot(/^Bomb/))
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(inventory({charges, onToggleBomb: toggle, bombArmed: true}))
        expect(slot(/^Bomb, Aim/).getAttribute("aria-pressed")).toBe("true")
    })

    it("fills the bank on a press, and says so when it is already full", () => {
        const use = vi.fn().mockReturnValueOnce(true).mockReturnValueOnce(false)
        render(inventory({charges: {...NO_CHARGES, refill: true}, onUseRefill: use}))

        fireEvent.click(slot(/^Refill/))
        expect(screen.queryByRole("button", {name: /Full/})).toBeNull()

        fireEvent.click(slot(/^Refill/))
        expect(use).toHaveBeenCalledTimes(2)
        expect(slot(/^Refill, Full/)).toBeTruthy()
    })

    it("only shows what it cannot use", () => {
        render(inventory({charges: {...NO_CHARGES, refill: true, bomb: true}, onUseRefill: undefined, onToggleBomb: undefined}))

        expect(slot(/^Refill/).disabled).toBe(true)
        expect(slot(/^Bomb/).disabled).toBe(true)
    })

    // The handle is the panel's bottom edge: over the slots it would hang in
    // mid-panel once folded, with the reading above it and nothing below.
    it("keeps the fold handle under the slots", () => {
        render(inventory())

        const section = screen.getByRole("region", {name: "Inventory"})
        expect(section.lastElementChild).toBe(screen.getByRole("button", {name: /Inventory/}))
    })

    it("folds, and stays folded on the next load", () => {
        render(inventory())

        fireEvent.click(screen.getByRole("button", {name: /Inventory/}))
        expect(screen.queryByRole("button", {name: /^Spread/})).toBeNull()
        expect(window.localStorage.getItem(INVENTORY_FOLDED_KEY)).toBe("1")

        cleanup()
        render(inventory())
        expect(screen.getByRole("button", {name: /Inventory/}).getAttribute("aria-expanded")).toBe("false")
    })
})
