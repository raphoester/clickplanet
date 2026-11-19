// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {act, cleanup, fireEvent, render, screen, within} from '@testing-library/react'
import Inventory, {DESKTOP_BAR_ROOM, InventoryProps, NOT_ENCLOSED_MS, NOTICE_MS} from './Inventory.tsx'
import {BUBBLE_MS} from '../components/Bubble.tsx'
import {ALL_OFF, BonusNoticeEvent, BonusRules, CHARGE_KINDS, Charges, Gates, NO_CHARGES, NO_GATES, NO_LEVEL, Switches, UnlockLevels} from "../../domain/bonus.ts"
import {BonusGuide, FRESH_GUIDE, LEARNING_USES} from "../../domain/bonusGuide.ts"
import {useBonusGuide} from "./bonusGuideStore.ts"
import {useBonus} from "./bonusStore.ts"
import {useScreen} from "../hud/screenStore.ts"
import {useLevel} from "../level/levelStore.ts"
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx"

afterEach(() => {
    cleanup()
    useBonusGuide.setState({guide: FRESH_GUIDE})
    useBonus.setState(useBonus.getInitialState(), true)
    useScreen.setState(useScreen.getInitialState(), true)
    useLevel.setState(useLevel.getInitialState(), true)
    vi.unstubAllGlobals()
})
beforeEach(() => window.localStorage.clear())

const RULES = {
    blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3, shields: 12, tileShields: 10, toll: [],
    signInRefill: false, enclosuresUnlimited: false, spreadUnlimited: false,
    reach: false, knockOut: false, capitalShieldsFixed: false, bombsOff: false, enclosuresOff: false, spreadOff: false, ralliesOff: false, virusInfections: 128,
    gatesOn: false, openKinds: [], refills: 1, bombs: 1, viruses: 1, mines: 3, mineRings: 2,
    rallies: 3, rallyRadius: 0.072, rallyLastsMs: 180_000, rallyCost: 0.5,
}

type Held = {
    guide?: BonusGuide
    notice?: BonusNoticeEvent
    charges?: Charges
    rules?: BonusRules
    switches?: Switches
    bombArmed?: boolean
    gates?: Gates
    level?: number
    virusAimed?: boolean
    mineAimed?: boolean
    rallyAimed?: boolean
}

function inventory({guide = FRESH_GUIDE, notice, charges = NO_CHARGES, rules = RULES, switches = ALL_OFF, bombArmed = false, virusAimed = false, mineAimed = false, rallyAimed = false, gates = NO_GATES, level = NO_LEVEL, ...props}: Partial<InventoryProps> & Held = {}) {
    act(() => {
        useBonusGuide.setState({guide})
        useBonus.setState({notice, charges, rules: {...rules, gatesOn: gates.on, openKinds: gates.open}, switches, bombArmed, virusAimed, mineAimed, rallyAimed})
        useLevel.setState({own: level === NO_LEVEL ? undefined : {level, xp: 0, levelXp: 0, nextLevelXp: 0}, unlocks: gates.unlocks})
    })
    return <Inventory onToggle={() => {}}
                      onToggleBomb={() => {}}
                      onUseRefill={() => true}
                      onToggleShield={() => {}}
                      {...props}/>
}

const slot = (name: RegExp) => screen.getByRole("button", {name}) as HTMLButtonElement

const all = () => screen.getByRole("button", {name: /^All bonuses/}) as HTMLButtonElement

const tray = () => screen.queryByRole("group", {name: "All bonuses"})

const inTray = (name: RegExp) => within(tray()!).getByRole("button", {name}) as HTMLButtonElement

const onBar = () => [...document.querySelectorAll(".inventory-slots > button")].map((button) => button.getAttribute("aria-label"))

const EVERY = {...NO_CHARGES, refills: 1, bombs: 1, spreadClicksLeft: 5, enclosures: 2, shields: 7}

const gated = (unlocks: UnlockLevels): Gates => ({on: true, open: CHARGE_KINDS.filter((kind) => !(kind in unlocks)), unlocks})

const bubble = () => screen.queryByRole("status")?.textContent

const learned: BonusGuide = {
    won: [],
    uses: {refill: LEARNING_USES, bomb: LEARNING_USES, spreadClicks: LEARNING_USES, encloseClicks: LEARNING_USES, shields: LEARNING_USES},
}

describe("Inventory", () => {
    it("shows every kind in the tray, and an empty one says how to get one instead of doing anything", () => {
        const toggle = vi.fn()
        const toggleBomb = vi.fn()
        render(inventory({onToggle: toggle, onToggleBomb: toggleBomb}))
        fireEvent.click(all())

        for (const name of [/^Refill/, /^Bomb/, /^Spread/, /^Enclose/, /^Shield/]) {
            expect(inTray(name).getAttribute("aria-disabled")).toBe("true")
        }

        fireEvent.click(inTray(/^Bomb/))
        fireEvent.click(inTray(/^Spread/))

        expect(toggle).not.toHaveBeenCalled()
        expect(toggleBomb).not.toHaveBeenCalled()
        expect(bubble()).toBe("Catch a ? box or win a quiz to get one")
    })

    it("puts the virus on the bar while one is held, and in the tray always", () => {
        render(inventory())
        expect(onBar()).toEqual(["All bonuses"])
        fireEvent.click(all())
        expect(inTray(/^Virus/).getAttribute("aria-disabled")).toBe("true")
        fireEvent.click(all())
        cleanup()

        const toggleVirus = vi.fn()
        render(inventory({charges: {...NO_CHARGES, viruses: 1}, onToggleVirus: toggleVirus}))
        fireEvent.click(slot(/^Virus/))

        expect(toggleVirus).toHaveBeenCalledOnce()
        expect(bubble()).toBe("Tap an enemy tile: the virus takes it, then spreads into their land on its own, up to 128 tiles")
    })

    it("says what a virus does on a hover, with its size once the rules are read", () => {
        const charges = {...NO_CHARGES, viruses: 1}
        render(inventory({charges, rules: {...RULES, virusInfections: 0}}))
        fireEvent.pointerEnter(slot(/^Virus/), {pointerType: "mouse"})
        expect(bubble()).toBe("Aim the virus at an enemy tile: it takes it, then spreads into their land on its own")
        cleanup()

        render(inventory({charges}))
        fireEvent.pointerEnter(slot(/^Virus/), {pointerType: "mouse"})
        expect(bubble()).toBe("Aim the virus at an enemy tile: it takes it, then spreads into their land on its own, up to 128 tiles")
    })

    it("says the virus is aimed, and how to put it away", () => {
        render(inventory({charges: {...NO_CHARGES, viruses: 1}, virusAimed: true}))

        expect(slot(/^Virus/).getAttribute("aria-label")).toBe("Virus, Aim")
        expect(slot(/^Virus/).getAttribute("aria-pressed")).toBe("true")
    })

    it("says a virus takes only an enemy tile with no shield, over its slot", () => {
        const charges = {...NO_CHARGES, viruses: 1}
        const {rerender} = render(inventory({charges}))

        rerender(inventory({charges, notice: {notice: "virusNoEnemy", seq: 1}}))

        expect(bubble()).toBe("A virus takes only an enemy tile with no shield")
    })

    it("counts the mines held, aims them and says how", () => {
        const toggleMine = vi.fn()
        render(inventory({charges: {...NO_CHARGES, mines: 2}, onToggleMine: toggleMine}))

        expect(slot(/^Mine/).getAttribute("aria-label")).toBe("Mine, 2/3, New")
        fireEvent.click(slot(/^Mine/))

        expect(toggleMine).toHaveBeenCalledOnce()
        expect(bubble()).toBe("Tap one of your tiles: if an enemy takes it, the mine explodes and takes the tiles around it")
    })

    it("says the mines are aimed, and that a mine goes only on the flag's tile with no mine", () => {
        const charges = {...NO_CHARGES, mines: 1}
        const {rerender} = render(inventory({charges, mineAimed: true}))
        expect(slot(/^Mine/).getAttribute("aria-pressed")).toBe("true")

        rerender(inventory({charges, mineAimed: true, notice: {notice: "mineNotYours", seq: 1}}))

        expect(bubble()).toBe("A mine goes only on your flag's tile with no mine")
    })

    it("says where the mine goes when a tap with a mine aimed took another flag's tile", () => {
        const charges = {...NO_CHARGES, mines: 1}
        const {rerender} = render(inventory({charges, mineAimed: true}))

        rerender(inventory({charges, mineAimed: true, notice: {notice: "mineTaken", seq: 1}}))

        expect(bubble()).toBe("Tile taken. Tap it again to lay a mine")
    })

    it("shows no bomb while bombs are off, held or not", () => {
        render(inventory({rules: {...RULES, bombsOff: true}, charges: {...NO_CHARGES, bombs: 1}}))
        fireEvent.click(all())

        expect(screen.queryByRole("button", {name: /^Bomb/})).toBeNull()
        expect(inTray(/^Refill/)).toBeDefined()
    })

    it("shows no enclose while enclosures are off, held or not", () => {
        render(inventory({rules: {...RULES, enclosuresOff: true}, charges: {...NO_CHARGES, enclosures: 2}}))
        fireEvent.click(all())

        expect(screen.queryByRole("button", {name: /^Enclose/})).toBeNull()
        expect(inTray(/^Spread/)).toBeDefined()
    })

    it("shows no spread while spreads are off, held or not", () => {
        render(inventory({rules: {...RULES, spreadOff: true}, charges: {...NO_CHARGES, spreadClicksLeft: 5}}))
        fireEvent.click(all())

        expect(screen.queryByRole("button", {name: /^Spread/})).toBeNull()
        expect(inTray(/^Enclose/)).toBeDefined()
    })

    it("shows no rally while rallies are off, held or not", () => {
        render(inventory({rules: {...RULES, ralliesOff: true}, charges: {...NO_CHARGES, rallies: 2}}))
        fireEvent.click(all())

        expect(screen.queryByRole("button", {name: /^Rally/})).toBeNull()
        expect(inTray(/^Spread/)).toBeDefined()
    })

    it("counts the pools against how many can be held", () => {
        render(inventory({charges: {...NO_CHARGES, spreadClicksLeft: 5, enclosures: 2, shields: 7}}))

        expect(slot(/^Spread, 5\/8/)).toBeTruthy()
        expect(slot(/^Enclose, 2\/3/)).toBeTruthy()
        expect(slot(/^Shield, 7\/12/)).toBeTruthy()
    })

    it("offers enclose with none held while enclosures are unlimited, and lets it be switched off", () => {
        const toggle = vi.fn()
        const unlimited = {...RULES, enclosuresUnlimited: true}
        const {container, rerender} = render(inventory({rules: unlimited, onToggle: toggle,
            switches: {...ALL_OFF, enclose: true}}))

        expect(slot(/^Enclose, ∞, On/).getAttribute("aria-disabled")).toBeNull()
        expect(container.querySelector(".inventory-loop")).toBeTruthy()

        fireEvent.click(slot(/^Enclose/))
        expect(toggle.mock.calls).toEqual([["enclose"]])

        rerender(inventory({rules: unlimited, onToggle: toggle}))
        expect(slot(/^Enclose, ∞$/).getAttribute("aria-pressed"), "no New: the game gave it, not a box").toBe("false")
        expect(container.querySelector(".inventory-loop")).toBeNull()
    })

    it("offers spread with none held while spreads are unlimited", () => {
        const toggle = vi.fn()
        const unlimited = {...RULES, spreadUnlimited: true}
        const {container} = render(inventory({rules: unlimited, onToggle: toggle, switches: {...ALL_OFF, spread: true}}))

        expect(slot(/^Spread, ∞, On/).getAttribute("aria-disabled")).toBeNull()
        expect(container.querySelector(".inventory-loop")).toBeTruthy()

        fireEvent.click(slot(/^Spread/))
        expect(toggle.mock.calls).toEqual([["spread"]])
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

        rerender(inventory({charges, switches, notice: {notice: "shieldCapital", seq: 3}}))
        expect(bubble()).toBe("No shield on a capital")
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
        const charges = {...NO_CHARGES, bombs: 1, shields: 2}
        const {rerender} = render(inventory({charges}))

        expect(slot(/^Bomb, New/).className).toContain("inventory-slot--new")
        expect(slot(/^Shield, 2\/12, New/)).toBeTruthy()

        rerender(inventory({charges, guide: {won: [], uses: {bomb: 1}}}))
        expect(slot(/^Bomb$/).className).not.toContain("inventory-slot--new")
        expect(slot(/^Shield, 2\/12, New/)).toBeTruthy()

        fireEvent.click(all())
        expect(inTray(/^Spread/).getAttribute("aria-label"), "none held").not.toContain("New")
    })

    it("says how a bonus works as it is switched on, for its first uses only", () => {
        vi.useFakeTimers()
        try {
            const charges = {...NO_CHARGES, enclosures: 2}
            const {rerender} = render(inventory({charges}))

            fireEvent.click(slot(/^Enclose/))
            expect(bubble()).toBe("Tap your tiles in a ring. The inside becomes yours, up to 25 tiles")
            expect(useBonusGuide.getState().guide.uses).toEqual({encloseClicks: 1})

            act(() => void vi.advanceTimersByTime(BUBBLE_MS))
            expect(bubble()).toBeUndefined()

            rerender(inventory({charges, guide: learned}))
            fireEvent.click(slot(/^Enclose/))
            expect(bubble()).toBeUndefined()
        } finally {
            vi.useRealTimers()
        }
    })

    it("says nothing and counts no use when a switch goes off", () => {
        const charges = {...NO_CHARGES, spreadClicksLeft: 3}
        render(inventory({charges, switches: {...ALL_OFF, spread: true}}))

        fireEvent.click(slot(/^Spread/))

        expect(bubble()).toBeUndefined()
        expect(useBonusGuide.getState().guide).toEqual(FRESH_GUIDE)
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
        render(inventory({charges: {...NO_CHARGES, bombs: 1}, guide: learned, gates: gated({shields: 3})}))

        expect(slot(/^Bomb/).getAttribute("aria-describedby")).toBeTruthy()
        expect(screen.getByText("Aim the bomb, then hold on the planet to drop it")).toBeTruthy()

        fireEvent.click(all())
        expect(within(tray()!).getByText("Aim the bomb, then hold on the planet to drop it")).toBeTruthy()
        expect(within(tray()!).getAllByText("Catch a ? box or win a quiz to get one")).toHaveLength(6)
        expect(within(tray()!).getByText("Unlocks at level 3")).toBeTruthy()
    })

    it("aims the bomb and puts it away", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, bombs: 1}
        const {rerender} = render(inventory({charges, onToggleBomb: toggle, guide: learned}))

        fireEvent.click(slot(/^Bomb/))
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(inventory({charges, onToggleBomb: toggle, guide: learned, bombArmed: true}))
        expect(slot(/^Bomb, Aim/).getAttribute("aria-pressed")).toBe("true")
    })

    it("counts the refills, bombs and viruses against how many a hand holds, once it holds more than one", () => {
        const toggle = vi.fn()
        const rules = {...RULES, refills: 3, bombs: 3, viruses: 2}
        const charges = {...NO_CHARGES, refills: 2, bombs: 2, viruses: 1}
        const {container, rerender} = render(inventory({rules, charges, onToggleBomb: toggle, guide: learned}))

        expect(slot(/^Refill/).getAttribute("aria-label")).toBe("Refill, 2/3")
        expect(slot(/^Bomb/).getAttribute("aria-label")).toBe("Bomb, 2/3")
        expect(slot(/^Virus/).getAttribute("aria-label")).toBe("Virus, 1/2")
        expect(container.querySelector(".inventory-slot--bomb .inventory-count")?.textContent).toBe("2/3")

        fireEvent.click(slot(/^Bomb/))
        expect(toggle).toHaveBeenCalledOnce()

        rerender(inventory({rules, charges: {...charges, bombs: 1}, onToggleBomb: toggle, guide: learned}))
        expect(slot(/^Bomb/).getAttribute("aria-label")).toBe("Bomb, 1/3")
        expect(slot(/^Bomb/).getAttribute("aria-disabled")).toBeNull()

        rerender(inventory({rules, charges: {...charges, bombs: 0}, onToggleBomb: toggle, guide: learned}))
        expect(screen.queryByRole("button", {name: /^Bomb/})).toBeNull()
        fireEvent.click(all())
        expect(inTray(/^Bomb, 0\/3/).getAttribute("aria-disabled")).toBe("true")
    })

    it("counts no refill, bomb or virus while a hand holds one at most, but several held all the same", () => {
        const {container, rerender} = render(inventory({charges: {...NO_CHARGES, refills: 1, bombs: 1, viruses: 1}, guide: learned}))

        expect(slot(/^Bomb/).getAttribute("aria-label")).toBe("Bomb")
        expect(container.querySelector(".inventory-slot--bomb .inventory-count")).toBeNull()
        expect(container.querySelector(".inventory-slot--refill .inventory-count")).toBeNull()

        rerender(inventory({charges: {...NO_CHARGES, bombs: 2}, guide: learned}))
        expect(slot(/^Bomb/).getAttribute("aria-label")).toBe("Bomb, 2/1")
    })

    it("aims a rally and puts it away, with what the hand holds of them", () => {
        const toggle = vi.fn()
        const charges = {...NO_CHARGES, rallies: 2}
        const {rerender} = render(inventory({charges, onToggleRally: toggle, guide: learned}))

        fireEvent.click(slot(/^Rally, 2\/3/))
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(inventory({charges, onToggleRally: toggle, guide: learned, rallyAimed: true}))
        expect(slot(/^Rally, 2\/3, Aim/).getAttribute("aria-pressed")).toBe("true")
        expect(screen.getByText("Put the rally away (Esc)")).toBeTruthy()
    })

    it("fills the bank on a press, and says so when it is already full", () => {
        const use = vi.fn().mockReturnValueOnce(true).mockReturnValueOnce(false)
        render(inventory({charges: {...NO_CHARGES, refills: 1}, onUseRefill: use, guide: learned}))

        fireEvent.click(slot(/^Refill/))
        expect(screen.queryByRole("button", {name: /Full/})).toBeNull()

        fireEvent.click(slot(/^Refill/))
        expect(use).toHaveBeenCalledTimes(2)
        expect(slot(/^Refill, Full/)).toBeTruthy()
        expect(bubble()).toBe("Your clicks are already full")
    })

    it("only shows what it cannot use", () => {
        render(inventory({
            charges: {...NO_CHARGES, refills: 1, bombs: 1, shields: 3},
            onUseRefill: undefined,
            onToggleBomb: undefined,
            onToggleShield: undefined,
        }))

        expect(slot(/^Refill/).disabled).toBe(true)
        expect(slot(/^Bomb/).disabled).toBe(true)
        expect(slot(/^Shield/).disabled).toBe(true)
    })

    it("keeps on the bar the kinds held, in the order they unlock, and All", () => {
        render(inventory({charges: {...NO_CHARGES, shields: 3, refills: 1}, guide: learned}))

        expect(onBar()).toEqual(["Refill", "Shield, 3/12", "All bonuses"])
        expect(tray()).toBeNull()
    })

    it("shows All alone when nothing is held", () => {
        render(inventory())

        expect(onBar()).toEqual(["All bonuses"])
    })

    it("tucks the held kinds the bar has no room for in the tray, and counts them on All", () => {
        render(inventory({charges: EVERY, guide: learned}))

        expect(onBar()).toHaveLength(DESKTOP_BAR_ROOM + 1)
        expect(onBar()).not.toContain("Shield, 7/12")
        expect(all().getAttribute("aria-label")).toBe("All bonuses, 1 more held")
        expect(all().textContent).toContain("+1")
    })

    it("keeps three on a phone's bar", () => {
        vi.stubGlobal("matchMedia", (query: string) => ({matches: true, media: query, addEventListener: () => {}, removeEventListener: () => {}}))
        render(inventory({charges: EVERY, guide: learned}))

        expect(onBar()).toEqual(["Refill", "Bomb", "Spread, 5/8", "All bonuses, 2 more held"])
    })

    it("counts the number held alone on a phone's bar, and against its size in the tray", () => {
        vi.stubGlobal("matchMedia", (query: string) => ({matches: true, media: query, addEventListener: () => {}, removeEventListener: () => {}}))
        const rules = {...RULES, refills: 3, shields: 60}
        const {container} = render(inventory({rules, charges: {...NO_CHARGES, refills: 1, shields: 26}, guide: learned}))

        expect(container.querySelector(".inventory-slots .inventory-slot--shields .inventory-count")?.textContent).toBe("26")
        expect(container.querySelector(".inventory-slots .inventory-slot--refill .inventory-count")?.textContent).toBe("1")
        expect(slot(/^Shield/).getAttribute("aria-label")).toBe("Shield, 26/60")

        fireEvent.click(all())
        expect(within(tray()!).getByRole("button", {name: /^Shield/}).querySelector(".inventory-count")?.textContent).toBe("26/60")
    })

    it("counts against the size on a desktop's bar", () => {
        const {container} = render(inventory({rules: {...RULES, shields: 60}, charges: {...NO_CHARGES, shields: 26}, guide: learned}))

        expect(container.querySelector(".inventory-slots .inventory-slot--shields .inventory-count")?.textContent).toBe("26/60")
    })

    it("counts ∞ on a phone's bar while spreads are unlimited", () => {
        vi.stubGlobal("matchMedia", (query: string) => ({matches: true, media: query, addEventListener: () => {}, removeEventListener: () => {}}))
        const {container} = render(inventory({rules: {...RULES, spreadUnlimited: true}, guide: learned}))

        expect(container.querySelector(".inventory-slots .inventory-slot--spreadClicks .inventory-count")?.textContent).toBe("∞")
    })

    it("says on All that a kind it holds is on", () => {
        render(inventory({charges: EVERY, guide: learned, switches: {...ALL_OFF, shield: true}}))

        expect(all().getAttribute("aria-label")).toBe("All bonuses, 1 more held, On")
        expect(all().className).toContain("inventory-slot--on")
    })

    it("calls All new while a kind it holds was never used", () => {
        render(inventory({charges: EVERY, guide: {won: [], uses: {...learned.uses, shields: 0}}}))

        expect(all().getAttribute("aria-label")).toBe("All bonuses, 1 more held, New")
    })

    it("lists every kind in the tray, in the order they unlock, a locked one with its level", () => {
        render(inventory({gates: gated({encloseClicks: 4, bomb: 5})}))

        fireEvent.click(all())
        expect(all().getAttribute("aria-expanded")).toBe("true")
        expect(within(tray()!).getAllByRole("button").map((button) => button.getAttribute("aria-label"))).toEqual([
            "Refill", "Spread, 0/8", "Shield, 0/12", "Virus", "Mine, 0/3", "Rally, 0/3", "Enclose, unlocks at level 4",
            "Bomb, unlocks at level 5",
        ])
        expect(inTray(/^Bomb/).className).toContain("inventory-slot--locked")
        expect(inTray(/^Bomb/).textContent).toContain("5")
    })

    it("says the level a locked kind unlocks at, and does nothing else", () => {
        const toggleBomb = vi.fn()
        render(inventory({gates: gated({bomb: 5}), onToggleBomb: toggleBomb}))

        fireEvent.click(all())
        fireEvent.click(inTray(/^Bomb/))

        expect(toggleBomb).not.toHaveBeenCalled()
        expect(bubble()).toBe("Unlocks at level 5")
        expect(tray()).not.toBeNull()
    })

    it("locks for now a gated kind the ladder names no level for", () => {
        render(inventory({gates: {on: true, open: ["refill", "spreadClicks", "shields"], unlocks: {encloseClicks: 4}}}))

        fireEvent.click(all())
        expect(inTray(/^Bomb/).getAttribute("aria-label")).toBe("Bomb, locked")
        fireEvent.click(inTray(/^Bomb/))
        expect(bubble()).toBe("Locked for now")
    })

    it("opens a gated kind at its level", () => {
        render(inventory({gates: gated({encloseClicks: 4, bomb: 5}), level: 4}))

        fireEvent.click(all())
        expect(inTray(/^Enclose/).getAttribute("aria-label")).toBe("Enclose, 0/3")
        expect(inTray(/^Bomb/).getAttribute("aria-label")).toBe("Bomb, unlocks at level 5")
    })

    it("locks nothing while the gates are off", () => {
        render(inventory({gates: {...gated({bomb: 5}), on: false}}))

        fireEvent.click(all())
        expect(inTray(/^Bomb/).getAttribute("aria-label")).toBe("Bomb")
    })

    it("plays a kind still held whatever the level", () => {
        const toggleBomb = vi.fn()
        render(inventory({gates: gated({bomb: 5}), charges: {...NO_CHARGES, bombs: 1}, onToggleBomb: toggleBomb, guide: learned}))

        fireEvent.click(slot(/^Bomb$/))

        expect(toggleBomb).toHaveBeenCalledTimes(1)
    })

    it("uses a kind pressed in the tray, closes it and says how it works over the bar", () => {
        const toggle = vi.fn()
        render(inventory({charges: EVERY, onToggleShield: toggle}))

        fireEvent.click(all())
        fireEvent.click(inTray(/^Shield/))

        expect(toggle).toHaveBeenCalledTimes(1)
        expect(tray()).toBeNull()
        expect(document.activeElement).toBe(all())
        expect(bubble()).toBe("Tap one of your tiles to shield it")
    })

    it("opens the tray in the dock's own slot, which floats over the dock", () => {
        const {container} = render(<ClickBudgetMeter>{inventory()}</ClickBudgetMeter>)

        fireEvent.click(all())

        expect(container.querySelector(".click-budget-dock > .click-budget-tray > .inventory-tray")).toBe(tray())
    })

    it("closes the tray on All, on Escape and on a press outside it", () => {
        render(<>{inventory()}<button>globe</button></>)

        fireEvent.click(all())
        fireEvent.click(all())
        expect(tray()).toBeNull()

        fireEvent.click(all())
        inTray(/^Shield/).focus()
        fireEvent.keyDown(document, {key: "Escape"})
        expect(tray()).toBeNull()
        expect(document.activeElement).toBe(all())

        fireEvent.click(all())
        fireEvent.pointerDown(inTray(/^Refill/))
        expect(tray()).not.toBeNull()
        fireEvent.pointerDown(screen.getByRole("button", {name: "globe"}))
        expect(tray()).toBeNull()
    })
})
