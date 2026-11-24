// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import {Decided} from "../../domain/flagSubmissions.ts"
import SubmissionNews, {SUBMISSION_NEWS_MS} from "./SubmissionNews.tsx"

const decided = (changes: Partial<Decided>): Decided => ({
    id: "s-1", name: "Bavaria", flag: "", state: "approved", submittedAt: 1, decidedAt: 2, paid: false, ...changes,
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("SubmissionNews", () => {
    it("says each decision in its own words", () => {
        const view = render(<SubmissionNews submission={decided({})} onDone={vi.fn()}/>)
        expect(screen.getByRole("status").textContent).toBe("Your flag Bavaria is approved!")

        view.rerender(<SubmissionNews submission={decided({state: "rejected"})} onDone={vi.fn()}/>)
        expect(screen.getByRole("status").textContent).toBe("Your flag Bavaria was rejected.")

        view.rerender(<SubmissionNews submission={decided({state: "opened", flag: "r-k3p9x2ma0b"})} onDone={vi.fn()}/>)
        expect(screen.getByRole("status").textContent).toBe("Bavaria is in the game!")

        view.rerender(<SubmissionNews submission={decided({state: "closed", flag: "r-k3p9x2ma0b"})} onDone={vi.fn()}/>)
        expect(screen.getByRole("status").textContent).toBe("Your flag Bavaria is closed.")
    })

    it("is done after a few seconds", () => {
        vi.useFakeTimers()
        const onDone = vi.fn()
        render(<SubmissionNews submission={decided({})} onDone={onDone}/>)

        act(() => vi.advanceTimersByTime(SUBMISSION_NEWS_MS))

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
