// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {VideoStory} from "../../domain/video/story.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import VideoMoment from "./VideoMoment.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

const story: VideoStory = {flag: "pl", country: "de", kind: "loss", tiles: 100, start: 60, end: 10}

const user = () => userEvent.setup({advanceTimers: vi.advanceTimersByTime})

function reveal() {
    act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
}

describe("VideoMoment", () => {
    it("says what happened in words of its own, not the video's headline", () => {
        render(<VideoMoment story={story} play={vi.fn()} onMake={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Call for backup! Your flag is losing ground in Germany"})).toBeDefined()
        expect(screen.getByText("The game makes a video of it in one tap. Once posted, its link brings new players to your flag.")).toBeDefined()
        expect(screen.queryByText(/POLAND IS LOSING/)).toBeNull()
    })

    it("offers the video, or not now, once the reveal has played", () => {
        render(<VideoMoment story={story} play={vi.fn()} onMake={vi.fn()} onClose={vi.fn()}/>)
        expect(screen.queryByRole("button")).toBeNull()

        reveal()

        expect(screen.getByRole("button", {name: "Make my video"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Not now"})).toBeDefined()
    })

    it("closes and makes the video", async () => {
        const onMake = vi.fn()
        const onClose = vi.fn()
        render(<VideoMoment story={story} play={vi.fn()} onMake={onMake} onClose={onClose}/>)
        reveal()

        await user().click(screen.getByRole("button", {name: "Make my video"}))

        expect(onClose).toHaveBeenCalledOnce()
        expect(onMake).toHaveBeenCalledOnce()
    })

    it("closes without the video on Not now", async () => {
        const onMake = vi.fn()
        const onClose = vi.fn()
        render(<VideoMoment story={story} play={vi.fn()} onMake={onMake} onClose={onClose}/>)
        reveal()

        await user().click(screen.getByRole("button", {name: "Not now"}))

        expect(onClose).toHaveBeenCalledOnce()
        expect(onMake).not.toHaveBeenCalled()
    })
})
