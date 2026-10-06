// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {act, cleanup, fireEvent, render, screen} from '@testing-library/react'
import Inventory, {InventoryProps} from './Inventory.tsx'
import {ALL_OFF, NO_CHARGES} from "../../domain/bonus.ts"

afterEach(cleanup)
beforeEach(() => window.localStorage.clear())

const rules = {blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3, defenders: 12, tileDefenders: 10, toll: []}

function inventory(props: Partial<InventoryProps> = {}) {
    return <Inventory charges={NO_CHARGES}
                      rules={rules}
                      switches={ALL_OFF}
                      onToggle={() => {}}
                      bombArmed={false}
                      onToggleBomb={() => {}}
                      onUseRefill={() => true}
                      onToggleDefend={() => {}}
                      {...props}/>
}

const slot = (name: RegExp) => screen.getByRole("button", {name}) as HTMLButtonElement

describe("Inventory", () => {
    it("shows every slot, and none can be pressed while nothing is held", () => {
        render(inventory())

        for (const name of [/^Refill/, /^Bomb/, /^Spread/, /^Enclose/, /^Defender/]) expect(slot(name).disabled).toBe(true)
    })

    it("counts the pools against how many can be held", () => {
        render(inventory({charges: {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2, defenders: 7}}))

        expect(slot(/^Spread, 5\/8/)).toBeTruthy()
        expect(slot(/^Enclose, 2\/3/)).toBeTruthy()
        expect(slot(/^Defender, 7\/12/)).toBeTruthy()
    })

    it("switches spread and enclose, and says which are on", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2}
        const {rerender} = render(inventory({charges, onToggle: toggle}))

        expect(slot(/^Spread/).getAttribute("aria-pressed")).toBe("false")
        fireEvent.click(slot(/^Spread/))
        fireEvent.click(slot(/^Enclose/))
        expect(toggle.mock.calls).toEqual([["spread"], ["enclose"]])

        rerender(inventory({charges, onToggle: toggle, switches: {...ALL_OFF, spread: true}}))
        expect(slot(/^Spread/).getAttribute("aria-pressed")).toBe("true")
        expect(slot(/^Spread, 5\/8, On/)).toBeTruthy()
        expect(slot(/^Enclose/).getAttribute("aria-pressed")).toBe("false")
    })

    it("switches the defender, and says when it is on", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, defenders: 5}
        const {rerender} = render(inventory({charges, onToggleDefend: toggle}))

        expect(slot(/^Defender/).getAttribute("aria-pressed")).toBe("false")
        fireEvent.click(slot(/^Defender/))
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(inventory({charges, onToggleDefend: toggle, switches: {...ALL_OFF, defend: true}}))
        expect(slot(/^Defender, 5\/12, On/).getAttribute("aria-pressed")).toBe("true")
    })

    it("says Full for a while each time a defender meets a full tile", () => {
        vi.useFakeTimers()
        try {
            const charges = {...NO_CHARGES, defenders: 5}
            const switches = {...ALL_OFF, defend: true}
            const {rerender} = render(inventory({charges, switches, garrisonFull: 0}))
            expect(slot(/^Defender, 5\/12, On/)).toBeTruthy()

            rerender(inventory({charges, switches, garrisonFull: 1}))
            expect(slot(/^Defender, 5\/12, Full/)).toBeTruthy()

            act(() => void vi.advanceTimersByTime(1500))
            rerender(inventory({charges, switches, garrisonFull: 2}))
            act(() => void vi.advanceTimersByTime(1500))
            expect(slot(/^Defender, 5\/12, Full/)).toBeTruthy()

            act(() => void vi.advanceTimersByTime(1000))
            expect(slot(/^Defender, 5\/12, On/)).toBeTruthy()
        } finally {
            vi.useRealTimers()
        }
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
        render(inventory({
            charges: {...NO_CHARGES, refill: true, bomb: true, defenders: 3},
            onUseRefill: undefined,
            onToggleBomb: undefined,
            onToggleDefend: undefined,
        }))

        expect(slot(/^Refill/).disabled).toBe(true)
        expect(slot(/^Bomb/).disabled).toBe(true)
        expect(slot(/^Defender/).disabled).toBe(true)
    })

    it("has no fold: the five slots are always one row", () => {
        render(inventory())

        const section = screen.getByRole("region", {name: "Inventory"})
        expect(section.querySelectorAll("button")).toHaveLength(5)
        expect(screen.queryByRole("button", {name: /Inventory/})).toBeNull()
    })
})
