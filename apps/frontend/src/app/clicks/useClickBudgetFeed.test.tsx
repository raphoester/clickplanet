// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {ClickBudget, ClickBudgetSource} from "../../backends/clickBudget.ts"
import {useClickBudget} from "./clickBudgetStore.ts"
import {useClickBudgetFeed} from "./useClickBudgetFeed.ts"

const reading: ClickBudget = {tokens: 4, capacity: 10, perSecond: 1, readAt: 0}

function watchable() {
    let tell: (budget: ClickBudget) => void = () => {}
    const stop = vi.fn()
    const source: ClickBudgetSource = {
        watchClickBudget: vi.fn((callback: (budget: ClickBudget) => void) => {
            tell = callback
            return stop
        }),
        priceFor: vi.fn(),
    }
    return {source, stop, tell: (budget: ClickBudget) => act(() => tell(budget))}
}

afterEach(() => {
    cleanup()
    useClickBudget.setState({budget: undefined})
})

describe("useClickBudgetFeed", () => {
    it("puts each reading in the store, and asks the price of the country played", () => {
        const {source, tell} = watchable()
        const {rerender} = renderHook(({country}) => useClickBudgetFeed(source, country), {initialProps: {country: "fr"}})

        tell(reading)
        expect(useClickBudget.getState().budget).toBe(reading)

        rerender({country: "bg"})
        expect(vi.mocked(source.priceFor).mock.calls).toEqual([["fr"], ["bg"]])
    })

    it("stops watching and empties the store once it stops", () => {
        const {source, stop, tell} = watchable()
        const {unmount} = renderHook(() => useClickBudgetFeed(source, "fr"))
        tell(reading)

        unmount()

        expect(stop).toHaveBeenCalledOnce()
        expect(useClickBudget.getState().budget).toBeUndefined()
    })
})
