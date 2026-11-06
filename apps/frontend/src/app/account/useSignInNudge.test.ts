// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, renderHook} from "@testing-library/react"
import {StrictMode} from "react"
import {useQuiz} from "../quiz/quizStore.ts"
import {NUDGE_AFTER_MS, useSignInNudge} from "./useSignInNudge.ts"

type Props = {offered: boolean, dryTaps: number, fastStart?: boolean}

function nudge(initialProps: Props = {offered: true, dryTaps: 0}, wrapper?: typeof StrictMode) {
    return renderHook((props: Props) => useSignInNudge(props.offered, props.dryTaps, props.fastStart ?? false),
        {initialProps, wrapper})
}

const wait = (ms: number) => act(() => {
    vi.advanceTimersByTime(ms)
})

const quizUp = (up: boolean) => act(() => {
    useQuiz.setState({state: up ? {phase: "offered", offer: {token: "t1", expiresAt: Infinity}} : {phase: "idle"}})
})

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    useQuiz.setState({state: {phase: "idle"}})
    vi.useRealTimers()
})

describe("useSignInNudge", () => {
    it("opens a moment after a dry tap", async () => {
        const {result, rerender} = nudge()
        expect(result.current.open).toBeUndefined()

        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS - 1)
        expect(result.current.open).toBeUndefined()
        await wait(1)

        expect(result.current.open).toBe("dry")
    })

    it("opens in StrictMode too", async () => {
        const {result, rerender} = nudge(undefined, StrictMode)

        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)

        expect(result.current.open).toBe("dry")
    })

    it("never opens over a quiz, and a quiz in the wait puts it off to the next dry tap", async () => {
        const {result, rerender} = nudge()

        quizUp(true)
        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBeUndefined()

        quizUp(false)
        rerender({offered: true, dryTaps: 2})
        await wait(NUDGE_AFTER_MS / 2)
        quizUp(true)
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBeUndefined()

        quizUp(false)
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open, "the quiz closing is no dry tap").toBeUndefined()

        rerender({offered: true, dryTaps: 3})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe("dry")
    })

    it("opens once a page", async () => {
        const {result, rerender} = nudge()
        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)

        act(() => result.current.close())
        rerender({offered: true, dryTaps: 2})
        await wait(NUDGE_AFTER_MS)

        expect(result.current.open).toBeUndefined()
    })

    it("does not open for a player it is not offered to", async () => {
        const {result, rerender} = nudge({offered: false, dryTaps: 0})

        rerender({offered: false, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBeUndefined()

        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open, "being offered it is no dry tap").toBeUndefined()
    })

    it("opens a moment after the fast start ends, and only then", async () => {
        const {result, rerender} = nudge({offered: true, dryTaps: 0, fastStart: false})

        rerender({offered: true, dryTaps: 0, fastStart: true})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open, "a fast start beginning is no reason").toBeUndefined()

        rerender({offered: true, dryTaps: 0, fastStart: false})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe("slowed")
    })

    it("waits out a quiz when the fast start ends", async () => {
        const {result, rerender} = nudge({offered: true, dryTaps: 0, fastStart: true})

        quizUp(true)
        rerender({offered: true, dryTaps: 0, fastStart: false})
        await wait(NUDGE_AFTER_MS * 4)
        expect(result.current.open).toBeUndefined()

        quizUp(false)
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe("slowed")
    })

    it("asks once on a dry tap and once when the fast start ends", async () => {
        const {result, rerender} = nudge({offered: true, dryTaps: 0, fastStart: true})

        rerender({offered: true, dryTaps: 1, fastStart: true})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe("dry")
        act(() => result.current.close())

        rerender({offered: true, dryTaps: 2, fastStart: false})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe("slowed")
        act(() => result.current.close())

        rerender({offered: true, dryTaps: 3, fastStart: true})
        rerender({offered: true, dryTaps: 4, fastStart: false})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBeUndefined()
    })

    it("does not ask a player it is not offered to when the fast start ends", async () => {
        const {result, rerender} = nudge({offered: false, dryTaps: 0, fastStart: true})

        rerender({offered: false, dryTaps: 0, fastStart: false})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBeUndefined()
    })
})
