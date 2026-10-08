import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {openStream, SILENCE_LIMIT_MS, Wakeups} from "./transport.ts"

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

function fakeWakeups() {
    let listener: (() => void) | undefined

    const wakeups: Wakeups = (wake) => {
        listener = wake
        return () => {
            listener = undefined
        }
    }

    return {
        wakeups,
        wake: () => listener?.(),
    }
}

describe("openStream", () => {
    beforeEach(() => vi.useFakeTimers())

    afterEach(() => {
        vi.useRealTimers()
        vi.restoreAllMocks()
    })

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
    it("tells the caller when a stream that was live comes back", async () => {
        const source = fakeSource<string>()
        const seen: string[] = []

        openStream(source.open, m => seen.push(m), "test", {onResumed: () => seen.push("resumed")})
        await settle()

        source.latest().push("one")
        await settle()
        source.latest().end()
        await vi.advanceTimersByTimeAsync(500)
        source.latest().push("two")
        await settle()

        expect(seen).toEqual(["one", "resumed", "two"])
    })

    it("does not call a stream that was never live resumed", async () => {
        const source = fakeSource<string>()
        const onResumed = vi.fn()

        openStream(source.open, () => {}, "test", {onResumed})
        await settle()

        source.latest().end()
        await vi.advanceTimersByTimeAsync(500)
        source.latest().push("first")
        await settle()

        expect(onResumed).not.toHaveBeenCalled()
    })

    it("reopens a stream that has gone silent", async () => {
        const source = fakeSource<string>()
        const onResumed = vi.fn()

        openStream(source.open, () => {}, "test", {onResumed, wakeups: fakeWakeups().wakeups})
        await settle()

        source.latest().push("alive")
        await vi.advanceTimersByTimeAsync(SILENCE_LIMIT_MS)

        expect(source.opened).toHaveLength(2)
        expect(source.opened[0].signal.aborted).toBe(true)

        source.latest().push("back")
        await settle()

        expect(onResumed).toHaveBeenCalledOnce()
    })

    it("keeps a stream that keeps hearing from the server", async () => {
        const source = fakeSource<string>()
        openStream(source.open, () => {}, "test", {wakeups: fakeWakeups().wakeups})
        await settle()

        for (let i = 0; i < 6; i++) {
            source.latest().push("heartbeat")
            await vi.advanceTimersByTimeAsync(30_000)
        }

        expect(source.opened).toHaveLength(1)
    })

    it("retries at once on a wake instead of waiting out the backoff", async () => {
        const source = fakeSource<string>()
        const page = fakeWakeups()
        openStream(source.open, () => {}, "test", {wakeups: page.wakeups})
        await settle()

        for (let i = 0; i < 5; i++) {
            source.latest().end()
            await vi.advanceTimersByTimeAsync(30_000)
        }
        source.latest().end()
        await settle()

        const before = source.opened.length
        page.wake()
        await settle()

        expect(source.opened).toHaveLength(before + 1)
    })

    it("reopens on a wake a stream that heard nothing while the page was frozen", async () => {
        const source = fakeSource<string>()
        const page = fakeWakeups()
        openStream(source.open, () => {}, "test", {wakeups: page.wakeups})
        await settle()

        source.latest().push("alive")
        await settle()
        vi.setSystemTime(Date.now() + SILENCE_LIMIT_MS)

        page.wake()
        await settle()

        expect(source.opened).toHaveLength(2)
        expect(source.opened[0].signal.aborted).toBe(true)
    })

    it("leaves a healthy stream alone on a wake", async () => {
        const source = fakeSource<string>()
        const page = fakeWakeups()
        openStream(source.open, () => {}, "test", {wakeups: page.wakeups})
        await settle()

        source.latest().push("alive")
        await settle()

        page.wake()
        await settle()

        expect(source.opened).toHaveLength(1)
    })

    it("stops listening for wakes once closed", async () => {
        const source = fakeSource<string>()
        const page = fakeWakeups()
        const close = openStream(source.open, () => {}, "test", {wakeups: page.wakeups})
        await settle()

        source.latest().end()
        await settle()
        close()

        page.wake()
        await settle()

        expect(source.opened).toHaveLength(1)
    })
})
