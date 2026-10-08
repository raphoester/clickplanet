// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {act, cleanup, fireEvent, render, screen} from '@testing-library/react'
import Inventory, {InventoryProps, NOT_ENCLOSED_MS, NOTICE_MS} from './Inventory.tsx'
import {BUBBLE_MS} from './Bubble.tsx'
import {ALL_OFF, NO_CHARGES} from "../../domain/bonus.ts"
import {BonusGuide, LEARNING_USES} from "../../domain/bonusGuide.ts"

afterEach(cleanup)
beforeEach(() => window.localStorage.clear())

const rules = {blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3, shields: 12, tileShields: 10, toll: []}

function inventory(props: Partial<InventoryProps> = {}) {
    return <Inventory charges={NO_CHARGES}
                      rules={rules}
                      switches={ALL_OFF}
                      onToggle={() => {}}
                      bombArmed={false}
                      onToggleBomb={() => {}}
                      onUseRefill={() => true}
                      onToggleShield={() => {}}
                      {...props}/>
}

const slot = (name: RegExp) => screen.getByRole("button", {name}) as HTMLButtonElement

const bubble = () => screen.queryByRole("status")?.textContent

const learned: BonusGuide = {
    won: [],
    uses: {refill: LEARNING_USES, bomb: LEARNING_USES, spreadClicks: LEARNING_USES, encloseClicks: LEARNING_USES, shields: LEARNING_USES},
}

describe("Inventory", () => {
    it("shows every slot, and an empty one says how to get one instead of doing anything", () => {
        const toggle = vi.fn()
        const toggleBomb = vi.fn()
        render(inventory({onToggle: toggle, onToggleBomb: toggleBomb}))

        for (const name of [/^Refill/, /^Bomb/, /^Spread/, /^Enclose/, /^Shield/]) {
            expect(slot(name).getAttribute("aria-disabled")).toBe("true")
        }

        fireEvent.click(slot(/^Bomb/))
        fireEvent.click(slot(/^Spread/))

        expect(toggle).not.toHaveBeenCalled()
        expect(toggleBomb).not.toHaveBeenCalled()
        expect(bubble()).toBe("Catch a ? box or win a quiz to get one")
    })

    it("counts the pools against how many can be held", () => {
        render(inventory({charges: {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2, shields: 7}}))

        expect(slot(/^Spread, 5\/8/)).toBeTruthy()
        expect(slot(/^Enclose, 2\/3/)).toBeTruthy()
        expect(slot(/^Shield, 7\/12/)).toBeTruthy()
    })

    it("switches spread and enclose, and says which are on", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2}
        const {rerender} = render(inventory({charges, onToggle: toggle, guide: learned}))

        expect(slot(/^Spread/).getAttribute("aria-pressed")).toBe("false")
        fireEvent.click(slot(/^Spread/))
        fireEvent.click(slot(/^Enclose/))
        expect(toggle.mock.calls).toEqual([["spread"], ["enclose"]])

        rerender(inventory({charges, onToggle: toggle, guide: learned, switches: {...ALL_OFF, spread: true}}))
        expect(slot(/^Spread/).getAttribute("aria-pressed")).toBe("true")
        expect(slot(/^Spread, 5\/8, On/)).toBeTruthy()
        expect(slot(/^Enclose/).getAttribute("aria-pressed")).toBe("false")
    })

    it("switches the shield, and says when it is on", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, shields: 5}
        const {rerender} = render(inventory({charges, onToggleShield: toggle, guide: learned}))

        expect(slot(/^Shield/).getAttribute("aria-pressed")).toBe("false")
        fireEvent.click(slot(/^Shield/))
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(inventory({charges, onToggleShield: toggle, guide: learned, switches: {...ALL_OFF, shield: true}}))
        expect(slot(/^Shield, 5\/12, On/).getAttribute("aria-pressed")).toBe("true")
    })

    it("says Full for a while each time a shield meets a full tile", () => {
        vi.useFakeTimers()
        try {
            const charges = {...NO_CHARGES, shields: 5}
            const switches = {...ALL_OFF, shield: true}
            const {rerender} = render(inventory({charges, switches, notice: undefined}))
            expect(slot(/^Shield, 5\/12, On/)).toBeTruthy()

            rerender(inventory({charges, switches, notice: {notice: "shieldFull", seq: 1}}))
            expect(slot(/^Shield, 5\/12, Full/)).toBeTruthy()
            expect(bubble()).toBe("That tile holds all the shields it can")

            act(() => void vi.advanceTimersByTime(NOTICE_MS - 500))
            rerender(inventory({charges, switches, notice: {notice: "shieldFull", seq: 2}}))
            act(() => void vi.advanceTimersByTime(NOTICE_MS - 500))
            expect(slot(/^Shield, 5\/12, Full/)).toBeTruthy()

            act(() => void vi.advanceTimersByTime(500))
            expect(slot(/^Shield, 5\/12, On/)).toBeTruthy()
            expect(bubble()).toBeUndefined()
        } finally {
            vi.useRealTimers()
        }
    })

    it("says where a shield goes when a click with shield on met another flag's tile", () => {
        const charges = {...NO_CHARGES, shields: 5}
        const switches = {...ALL_OFF, shield: true}
        const {rerender} = render(inventory({charges, switches}))

        rerender(inventory({charges, switches, notice: {notice: "shieldTaken", seq: 1}}))
        expect(bubble()).toBe("Tile taken. Tap it again to shield it")

        rerender(inventory({charges, switches, notice: {notice: "shieldNotYours", seq: 2}}))
        expect(bubble()).toBe("Shields go on your own tiles")
    })

    it("says so, for a while, when an enclose click closed nothing", () => {
        vi.useFakeTimers()
        try {
            const charges = {...NO_CHARGES, enclosures: 2}
            const switches = {...ALL_OFF, enclose: true}
            const {rerender} = render(inventory({charges, switches}))

            rerender(inventory({charges, switches, notice: {notice: "nothingEnclosed", seq: 1}}))
            expect(bubble()).toBe("Shape is not closed or is too big (25 tiles max)")

            act(() => void vi.advanceTimersByTime(NOT_ENCLOSED_MS))
            expect(bubble()).toBeUndefined()
        } finally {
            vi.useRealTimers()
        }
    })

    it("lets a slot's line go once its charge is spent", () => {
        const switches = {...ALL_OFF, shield: true}
        const {rerender} = render(inventory({charges: {...NO_CHARGES, shields: 5}, switches}))

        rerender(inventory({charges: {...NO_CHARGES, shields: 5}, switches, notice: {notice: "shieldTaken", seq: 1}}))
        rerender(inventory({charges: {...NO_CHARGES, shields: 5, spreadClicksLeft: 2}, switches, notice: {notice: "shieldTaken", seq: 1}}))
        expect(bubble()).toBe("Tile taken. Tap it again to shield it")

        rerender(inventory({charges: {...NO_CHARGES, shields: 4, spreadClicksLeft: 2}, switches, notice: {notice: "shieldTaken", seq: 1}}))
        expect(bubble()).toBeUndefined()
    })

    it("says nothing for a notice it was mounted with", () => {
        render(inventory({charges: {...NO_CHARGES, shields: 5}, notice: {notice: "shieldNotYours", seq: 4}}))

        expect(bubble()).toBeUndefined()
    })

    it("calls a bonus never used new, until it is used", () => {
        const charges = {...NO_CHARGES, bomb: true, shields: 2}
        const {rerender} = render(inventory({charges}))

        expect(slot(/^Bomb, New/).className).toContain("inventory-slot--new")
        expect(slot(/^Shield, 2\/12, New/)).toBeTruthy()
        expect(slot(/^Spread/).getAttribute("aria-label")).not.toContain("New")

        rerender(inventory({charges, guide: {won: [], uses: {bomb: 1}}}))
        expect(slot(/^Bomb$/).className).not.toContain("inventory-slot--new")
        expect(slot(/^Shield, 2\/12, New/)).toBeTruthy()
    })

    it("says how a bonus works as it is switched on, for its first uses only", () => {
        vi.useFakeTimers()
        try {
            const used = vi.fn()
            const charges = {...NO_CHARGES, enclosures: 2}
            const {rerender} = render(inventory({charges, onUsed: used}))

            fireEvent.click(slot(/^Enclose/))
            expect(bubble()).toBe("Tap your tiles in a ring. The inside becomes yours, up to 25 tiles")
            expect(used).toHaveBeenCalledWith("encloseClicks")

            act(() => void vi.advanceTimersByTime(BUBBLE_MS))
            expect(bubble()).toBeUndefined()

            rerender(inventory({charges, onUsed: used, guide: learned}))
            fireEvent.click(slot(/^Enclose/))
            expect(bubble()).toBeUndefined()
        } finally {
            vi.useRealTimers()
        }
    })

    it("says nothing and counts no use when a switch goes off", () => {
        const used = vi.fn()
        const charges = {...NO_CHARGES, spreadClicksLeft: 3}
        render(inventory({charges, onUsed: used, switches: {...ALL_OFF, spread: true}}))

        fireEvent.click(slot(/^Spread/))

        expect(bubble()).toBeUndefined()
        expect(used).not.toHaveBeenCalled()
    })

    it("says what a slot does while a mouse rests on it, and not on a touch", () => {
        render(inventory({charges: {...NO_CHARGES, shields: 2}, guide: learned}))

        fireEvent.pointerEnter(slot(/^Shield/), {pointerType: "touch"})
        expect(bubble()).toBeUndefined()

        fireEvent.pointerEnter(slot(/^Shield/), {pointerType: "mouse"})
        expect(bubble()).toBe("Switch shield on, then tap your tiles to shield them")

        fireEvent.pointerLeave(slot(/^Shield/), {pointerType: "mouse"})
        expect(bubble()).toBeUndefined()
    })

    it("describes each slot to a screen reader", () => {
        render(inventory({charges: {...NO_CHARGES, bomb: true}, guide: learned}))

        expect(slot(/^Bomb/).getAttribute("aria-describedby")).toBeTruthy()
        expect(screen.getByText("Aim the bomb, then hold on the planet to drop it")).toBeTruthy()
        expect(screen.getAllByText("Catch a ? box or win a quiz to get one")).toHaveLength(4)
    })

    it("aims the bomb and puts it away", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, bomb: true}
        const {rerender} = render(inventory({charges, onToggleBomb: toggle, guide: learned}))

        fireEvent.click(slot(/^Bomb/))
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(inventory({charges, onToggleBomb: toggle, guide: learned, bombArmed: true}))
        expect(slot(/^Bomb, Aim/).getAttribute("aria-pressed")).toBe("true")
    })

    it("fills the bank on a press, and says so when it is already full", () => {
        const use = vi.fn().mockReturnValueOnce(true).mockReturnValueOnce(false)
        render(inventory({charges: {...NO_CHARGES, refill: true}, onUseRefill: use, guide: learned}))

        fireEvent.click(slot(/^Refill/))
        expect(screen.queryByRole("button", {name: /Full/})).toBeNull()

        fireEvent.click(slot(/^Refill/))
        expect(use).toHaveBeenCalledTimes(2)
        expect(slot(/^Refill, Full/)).toBeTruthy()
        expect(bubble()).toBe("Your clicks are already full")
    })

    it("only shows what it cannot use", () => {
        render(inventory({
            charges: {...NO_CHARGES, refill: true, bomb: true, shields: 3},
            onUseRefill: undefined,
            onToggleBomb: undefined,
            onToggleShield: undefined,
        }))

        expect(slot(/^Refill/).disabled).toBe(true)
        expect(slot(/^Bomb/).disabled).toBe(true)
        expect(slot(/^Shield/).disabled).toBe(true)
    })

    it("has no fold: the five slots are always one row", () => {
        render(inventory())

        const section = screen.getByRole("region", {name: "Inventory"})
        expect(section.querySelectorAll("button")).toHaveLength(5)
        expect(screen.queryByRole("button", {name: /Inventory/})).toBeNull()
    })
})
