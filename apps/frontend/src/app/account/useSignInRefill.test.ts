// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, renderHook} from "@testing-library/react"
import {StrictMode} from "react"
import {SIGN_IN_REFILL_WAIT_MS, useSignInRefill} from "./useSignInRefill.ts"

type Props = {due: boolean, held: boolean}

function refill(initialProps: Props, wrapper?: typeof StrictMode) {
    const refresh = vi.fn()
    const spend = vi.fn()
    const hook = renderHook((props: Props) => useSignInRefill(props.due, props.held, refresh, spend), {initialProps, wrapper})
    return {...hook, refresh, spend}
}

const wait = (ms: number) => act(() => {
    vi.advanceTimersByTime(ms)
})

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    vi.useRealTimers()
})

describe("useSignInRefill", () => {
    it("reads the charges until the refill lands, then spends it once", async () => {
        const {rerender, refresh, spend} = refill({due: false, held: false})
        expect(refresh).not.toHaveBeenCalled()

        rerender({due: true, held: false})
        await wait(0)
        expect(refresh).toHaveBeenCalledTimes(1)
        await wait(2_000)
        expect(refresh).toHaveBeenCalledTimes(2)
        expect(spend).not.toHaveBeenCalled()

        rerender({due: true, held: true})
        rerender({due: true, held: false})
        rerender({due: true, held: true})

        expect(spend).toHaveBeenCalledTimes(1)
    })

    it("spends once in StrictMode too", async () => {
        const {rerender, spend} = refill({due: false, held: false}, StrictMode)

        rerender({due: true, held: true})
        await wait(0)

        expect(spend).toHaveBeenCalledTimes(1)
    })

    it("leaves a refill that lands after the wait to the player", async () => {
        const {rerender, refresh, spend} = refill({due: true, held: false})

        await wait(SIGN_IN_REFILL_WAIT_MS)
        rerender({due: true, held: true})

        expect(refresh).toHaveBeenCalledTimes(3)
        expect(spend).not.toHaveBeenCalled()
    })

    it("does nothing for a sign-in that earned none", async () => {
        const {refresh, spend} = refill({due: false, held: true})

        await wait(SIGN_IN_REFILL_WAIT_MS)

        expect(refresh).not.toHaveBeenCalled()
        expect(spend).not.toHaveBeenCalled()
    })
})
