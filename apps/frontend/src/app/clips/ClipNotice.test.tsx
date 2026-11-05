// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Appearance} from "../../backends/clips.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {TIKTOK_PROFILE} from "../../links.ts"
import ClipNotice from "./ClipNotice.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

const appearance: Appearance = {
    postId: "post-1",
    title: "FRANCE VS TURKEY",
    scheduledAt: 1,
    links: [{platform: "tiktok", url: ""}, {platform: "youtube", url: "https://www.youtube.com/shorts/abc"}],
}

const user = () => userEvent.setup({advanceTimers: vi.advanceTimersByTime})

function reveal() {
    act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
}

describe("ClipNotice", () => {
    it("says the player is in a clip, and names the clip", () => {
        render(<ClipNotice appearance={appearance} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "You are in a clip! FRANCE VS TURKEY"})).toBeDefined()
    })

    it("plays the fanfare as it opens", () => {
        const play = vi.fn()
        render(<ClipNotice appearance={appearance} play={play} onClose={vi.fn()}/>)

        expect(play).toHaveBeenCalledWith("title")
    })

    it("offers to watch it on each platform, in a new tab, once the reveal has played", () => {
        render(<ClipNotice appearance={appearance} play={vi.fn()} onClose={vi.fn()}/>)
        expect(screen.queryByRole("link")).toBeNull()

        reveal()

        const tiktok = screen.getByRole("link", {name: "Watch on TikTok"})
        expect(tiktok.getAttribute("href")).toBe(TIKTOK_PROFILE)
        expect(tiktok.getAttribute("target")).toBe("_blank")
        expect(screen.getByRole("link", {name: "Watch on YouTube"}).getAttribute("href")).toBe("https://www.youtube.com/shorts/abc")
        expect(screen.getByRole("button", {name: "Close"})).toBeDefined()
    })

    it("closes once the player goes to watch it", async () => {
        const onClose = vi.fn()
        render(<ClipNotice appearance={appearance} play={vi.fn()} onClose={onClose}/>)
        reveal()

        await user().click(screen.getByRole("link", {name: "Watch on TikTok"}))

        expect(onClose).toHaveBeenCalled()
    })

    it("closes on Escape only once its buttons are there", async () => {
        const onClose = vi.fn()
        render(<ClipNotice appearance={appearance} play={vi.fn()} onClose={onClose}/>)

        await user().keyboard("{Escape}")
        expect(onClose).not.toHaveBeenCalled()

        reveal()
        await user().keyboard("{Escape}")
        expect(onClose).toHaveBeenCalled()
    })
})
