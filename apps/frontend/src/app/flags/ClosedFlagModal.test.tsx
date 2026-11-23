// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, renderHook, screen} from "@testing-library/react"
import ClosedFlagModal from "./ClosedFlagModal.tsx"
import {useHeldFlagClosings} from "./useHeldFlagClosings.ts"
import {CONFIRM_ARM_MS} from "../knockout/OutModal.tsx"
import {useFlags} from "./flagsStore.ts"
import type {ListedFlag} from "../../domain/flags.ts"
import type {AllegianceSource} from "../../backends/backend.ts"

const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}
const BAVARIA: ListedFlag = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: true, sold: {picture: PICTURE, endsAt: 1, closedAt: 2}}
const SAVOY: ListedFlag = {code: "r-0a1b2c3d4e", name: "Savoy", closed: false, sold: {picture: PICTURE, endsAt: 3}}

beforeEach(() => {
    vi.useFakeTimers({toFake: ["setTimeout", "clearTimeout"]})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    useFlags.getState().serve(undefined)
})

const click = (target: Element) => act(async () => { fireEvent.click(target) })
const optionOf = (code: string) => document.querySelector<HTMLElement>(`[role=option][id="country-option-${code}"]`)

describe("the window of a closed flag", () => {
    it("asks a player whose flag closed to pick another, from the open ones alone, with no way to close it", async () => {
        useFlags.getState().serve({listFlags: async () => [BAVARIA, SAVOY]})
        await act(async () => { await Promise.resolve() })
        const onPick = vi.fn(async () => {})
        render(<ClosedFlagModal allegiance={{countryId: BAVARIA.code, closed: true}} onPick={onPick}/>)

        expect(screen.getByRole("dialog", {name: "Bavaria is closed"})).toBeDefined()
        expect(screen.getByText("Pick another flag.")).toBeDefined()
        expect(optionOf(BAVARIA.code)).toBeNull()
        expect(optionOf("fr")).not.toBeNull()
        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()

        await click(optionOf(SAVOY.code)!)
        act(() => { vi.advanceTimersByTime(CONFIRM_ARM_MS) })
        await click(screen.getByRole("button", {name: "Confirm"}))
        expect(onPick).toHaveBeenCalledWith({code: SAVOY.code})
    })

    it("shows nothing while the flag held is open, or kept playable by a lock", () => {
        const {container} = render(<ClosedFlagModal allegiance={{countryId: SAVOY.code, locked: true}} onPick={vi.fn()}/>)
        expect(container.innerHTML).toBe("")
        expect(screen.queryByRole("dialog")).toBeNull()
    })
})

describe("the flag held, read again", () => {
    it("when the flags' stream tells it closed or back, and not for another flag", async () => {
        useFlags.getState().serve({listFlags: async () => [{...BAVARIA, closed: false}]})
        await act(async () => { await Promise.resolve() })
        const source: AllegianceSource = {listenForAllegiance: () => () => {}, pledgeAllegiance: vi.fn(), refreshAllegiance: vi.fn(async () => {})}
        renderHook(() => useHeldFlagClosings(source, BAVARIA.code))
        expect(source.refreshAllegiance).not.toHaveBeenCalled()

        act(() => useFlags.getState().told({...SAVOY, closed: true}))
        expect(source.refreshAllegiance).not.toHaveBeenCalled()

        act(() => useFlags.getState().told(BAVARIA))
        expect(source.refreshAllegiance).toHaveBeenCalledTimes(1)

        act(() => useFlags.getState().told({...BAVARIA, closed: false}))
        expect(source.refreshAllegiance).toHaveBeenCalledTimes(2)
    })
})
