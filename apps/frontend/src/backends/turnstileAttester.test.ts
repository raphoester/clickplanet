// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {turnstileAttester} from "./turnstileSession.ts"

type Options = {
    callback: (token: string) => void
    "error-callback": (code?: string) => void
    "before-interactive-callback": () => void
    "after-interactive-callback": () => void
}

let rendered: Options[] = []
const remove = vi.fn()

beforeEach(() => {
    vi.useFakeTimers()
    rendered = []
    remove.mockClear()
    window.turnstile = {
        render: (_container: HTMLElement, options: unknown) => {
            rendered.push(options as Options)
            return `widget-${rendered.length}`
        },
        remove,
    }
})

afterEach(() => {
    vi.useRealTimers()
})

function host(): HTMLElement {
    const found = document.querySelector<HTMLElement>(".turnstile-host")
    expect(found).not.toBeNull()
    return found as HTMLElement
}

async function attesting(): Promise<{outcome: () => string, widget: Options}> {
    let outcome = "pending"
    turnstileAttester("sitekey", "session")().then(
        (token) => outcome = `token ${token}`,
        (e: Error) => outcome = `refused: ${e.message}`,
    )
    await vi.advanceTimersByTimeAsync(0)

    return {outcome: () => outcome, widget: rendered[rendered.length - 1]}
}

describe("turnstileAttester", () => {
    it("fills the screen with the box while it asks for a tick, and clears it once ticked", async () => {
        const {outcome, widget} = await attesting()
        expect(host().classList.contains("asking")).toBe(false)

        widget["before-interactive-callback"]()
        expect(host().classList.contains("asking")).toBe(true)

        widget["after-interactive-callback"]()
        expect(host().classList.contains("asking")).toBe(false)

        widget.callback("tok")
        await vi.advanceTimersByTimeAsync(0)
        expect(outcome()).toBe("token tok")
    })

    it("gives a person 5 minutes to tick, not the 30 s of an invisible check", async () => {
        const {outcome, widget} = await attesting()
        widget["before-interactive-callback"]()

        await vi.advanceTimersByTimeAsync(4 * 60_000)
        expect(outcome()).toBe("pending")
        expect(host().classList.contains("asking")).toBe(true)

        await vi.advanceTimersByTimeAsync(60_000)
        expect(outcome()).toBe("refused: Turnstile did not answer in time")
        expect(host().classList.contains("asking")).toBe(false)
        expect(remove).toHaveBeenCalledWith("widget-1")
    })

    it("still ends an invisible check at 30 s", async () => {
        const {outcome} = await attesting()

        await vi.advanceTimersByTimeAsync(30_000)

        expect(outcome()).toBe("refused: Turnstile did not answer in time")
    })

    it("clears the screen when the box fails", async () => {
        const {outcome, widget} = await attesting()
        widget["before-interactive-callback"]()

        widget["error-callback"]("600010")
        await vi.advanceTimersByTimeAsync(0)

        expect(outcome()).toBe("refused: Turnstile failed: 600010")
        expect(host().classList.contains("asking")).toBe(false)
    })

    it("keeps the screen filled while a second box still asks", async () => {
        const first = await attesting()
        const second = await attesting()
        first.widget["before-interactive-callback"]()
        second.widget["before-interactive-callback"]()

        first.widget.callback("one")
        await vi.advanceTimersByTimeAsync(0)
        expect(host().classList.contains("asking")).toBe(true)

        second.widget.callback("two")
        await vi.advanceTimersByTimeAsync(0)
        expect(host().classList.contains("asking")).toBe(false)
    })
})
