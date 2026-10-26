// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {StrictMode} from "react"
import {useTerritoryCard} from "./useTerritoryCard.ts"
import {LEAVE_GRACE_MS} from "../players/useTileHolder.ts"
import {SAMPLE_MS} from "../viewer/useLeaderboardFeed.ts"
import {Territory, TerritoryPeek} from "../../domain/territory.ts"

const SICILY: Territory = {
    landmass: 12, name: "Sicily", ground: "it", size: 100,
    holdings: [{flag: "fr", tiles: 60}], fortifiedBy: undefined, target: undefined,
}

const PEEK: TerritoryPeek = {landmass: 12, point: {x: 300, y: 200}, touch: false}

type Pointing = {peek: TerritoryPeek, id: number} | undefined

function globe(territory = SICILY) {
    const state = {territory}
    return {
        state,
        territoryOf: vi.fn((landmass: number) => landmass === state.territory.landmass ? state.territory : undefined),
        showTerritory: vi.fn(),
    }
}

function render(planet: ReturnType<typeof globe>) {
    return renderHook(
        ({pointing}) => useTerritoryCard(pointing, planet.territoryOf, planet.showTerritory),
        {initialProps: {pointing: {peek: PEEK, id: 1} as Pointing}, wrapper: StrictMode},
    )
}

describe("useTerritoryCard", () => {
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("shows the territory pointed at, beside the pointer, and highlights it on the globe", () => {
        const planet = globe()

        const {result} = render(planet)

        expect(result.current.shown).toEqual({territory: SICILY, point: PEEK.point, touch: false, id: 1})
        expect(planet.showTerritory).toHaveBeenLastCalledWith(12)
    })

    it("follows the territory while its card is open, and keeps the same card while nothing changed", () => {
        vi.useFakeTimers()
        const planet = globe()
        const {result} = render(planet)
        const first = result.current.shown

        act(() => void vi.advanceTimersByTime(SAMPLE_MS))
        expect(result.current.shown).toBe(first)

        planet.state.territory = {...SICILY, holdings: [{flag: "fr", tiles: 61}]}
        act(() => void vi.advanceTimersByTime(SAMPLE_MS))
        expect(result.current.shown?.territory.holdings).toEqual([{flag: "fr", tiles: 61}])
        expect(result.current.shown?.point).toEqual(PEEK.point)
    })

    it("goes a moment after the pointer leaves, unless the pointer is on the card, and takes the highlight with it", () => {
        vi.useFakeTimers()
        const planet = globe()
        const {result, rerender} = render(planet)

        act(() => result.current.hold(true))
        rerender({pointing: undefined})
        act(() => void vi.advanceTimersByTime(LEAVE_GRACE_MS * 2))
        expect(result.current.shown).toBeDefined()

        act(() => result.current.hold(false))
        act(() => void vi.advanceTimersByTime(LEAVE_GRACE_MS))
        expect(result.current.shown).toBeUndefined()
        expect(planet.showTerritory).toHaveBeenLastCalledWith(undefined)
    })

    it("closes at once when dismissed", () => {
        const planet = globe()
        const {result} = render(planet)

        act(() => result.current.dismiss())

        expect(result.current.shown).toBeUndefined()
        expect(planet.showTerritory).toHaveBeenLastCalledWith(undefined)
    })
})
