import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {openStream} from "./transport.ts"

type Opened<T> = {
    signal: AbortSignal
    push: (value: T) => void
    end: () => void
    fail: (error: unknown) => void
}

function fakeSource<T>() {
    const opened: Opened<T>[] = []

    const open = (signal: AbortSignal): AsyncIterable<T> => {
        const queue: T[] = []
        let wake: (() => void) | undefined
        let done = false
        let failure: unknown

        const nudge = () => {
            wake?.()
            wake = undefined
        }

        const handle: Opened<T> = {
            signal,
            push: (value) => {
                queue.push(value)
                nudge()
            },
            end: () => {
                done = true
                nudge()
            },
            fail: (error) => {
                failure = error
                done = true
                nudge()
            },
        }

        opened.push(handle)
        signal.addEventListener("abort", handle.end)

        return {
            async* [Symbol.asyncIterator]() {
                for (; ;) {
                    while (queue.length > 0) yield queue.shift() as T
                    if (failure !== undefined) throw failure
                    if (done) return
                    await new Promise<void>(resolve => {
                        wake = resolve
                    })
                }
            },
        }
    }

    return {
        open,
        opened,
        latest: () => opened[opened.length - 1],
    }
}

describe("openStream", () => {
    beforeEach(() => vi.useFakeTimers())

    afterEach(() => {
        vi.useRealTimers()
        vi.restoreAllMocks()
    })

    // The iteration runs on microtasks, so a turn has to be handed back before
    // the reconnect a stream's end schedules can be observed.
    const settle = () => vi.advanceTimersByTimeAsync(0)

    it("opens once and forwards what arrives", async () => {
        const source = fakeSource<string>()
        const received: string[] = []

        openStream(source.open, m => received.push(m), "test")
        await settle()

        expect(source.opened).toHaveLength(1)

        source.latest().push("one")
        source.latest().push("two")
        await settle()

        expect(received).toEqual(["one", "two"])
    })

    it("reopens after the stream ends", async () => {
        const source = fakeSource<string>()
        openStream(source.open, () => {}, "test")
        await settle()

        source.latest().end()
        await vi.advanceTimersByTimeAsync(500)

        expect(source.opened).toHaveLength(2)
    })

    it("reopens after the stream fails", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})

        const source = fakeSource<string>()
        openStream(source.open, () => {}, "test")
        await settle()

        source.latest().fail(new Error("connection reset"))
        await vi.advanceTimersByTimeAsync(500)

        expect(source.opened).toHaveLength(2)
    })

    it("backs off exponentially while the backend stays down", async () => {
        const source = fakeSource<string>()
        openStream(source.open, () => {}, "test")
        await settle()

        const delays = [500, 1000, 2000, 4000]
        for (const [i, delay] of delays.entries()) {
            source.latest().end()
            await settle()

            await vi.advanceTimersByTimeAsync(delay - 1)
            expect(source.opened, `retry ${i} fired early`).toHaveLength(i + 1)

            await vi.advanceTimersByTimeAsync(1)
            expect(source.opened, `retry ${i} did not fire`).toHaveLength(i + 2)
        }
    })

    it("caps the backoff", async () => {
        const source = fakeSource<string>()
        openStream(source.open, () => {}, "test")
        await settle()

        for (let i = 0; i < 20; i++) {
            source.latest().end()
            await vi.advanceTimersByTimeAsync(30_000)
        }

        const before = source.opened.length
        source.latest().end()
        await vi.advanceTimersByTimeAsync(30_000)

        expect(source.opened).toHaveLength(before + 1)
    })

    it("resets the backoff once a message has come down the stream", async () => {
        const source = fakeSource<string>()
        openStream(source.open, () => {}, "test")
        await settle()

        source.latest().end()
        await vi.advanceTimersByTimeAsync(500)
        source.latest().end()
        await vi.advanceTimersByTimeAsync(1000)

        source.latest().push("alive")
        await settle()

        const before = source.opened.length
        source.latest().end()
        await vi.advanceTimersByTimeAsync(500)

        expect(source.opened).toHaveLength(before + 1)
    })

    it("aborts the open stream when the caller stops listening", async () => {
        const source = fakeSource<string>()
        const close = openStream(source.open, () => {}, "test")
        await settle()

        expect(source.latest().signal.aborted).toBe(false)

        close()
        expect(source.latest().signal.aborted).toBe(true)
    })

    it("does not reopen after the caller stops listening", async () => {
        const source = fakeSource<string>()
        const close = openStream(source.open, () => {}, "test")
        await settle()

        close()
        await vi.advanceTimersByTimeAsync(60_000)

        expect(source.opened).toHaveLength(1)
    })

    it("cancels a reopen that was already scheduled", async () => {
        const source = fakeSource<string>()
        const close = openStream(source.open, () => {}, "test")
        await settle()

        source.latest().end()
        await settle()
        close()

        await vi.advanceTimersByTimeAsync(60_000)

        expect(source.opened).toHaveLength(1)
    })

    it("does not report the caller's own cancellation as a failure", async () => {
        const errors = vi.spyOn(console, "error").mockImplementation(() => {})

        const source = fakeSource<string>()
        const close = openStream(source.open, () => {}, "test")
        await settle()

        source.latest().fail(new Error("aborted"))
        close()
        await vi.advanceTimersByTimeAsync(60_000)

        expect(errors).not.toHaveBeenCalled()
        expect(source.opened).toHaveLength(1)
    })
})
