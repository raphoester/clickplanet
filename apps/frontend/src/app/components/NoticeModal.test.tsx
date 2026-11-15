// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import NoticeModal, {NOTICE_HOLD_MS} from "./NoticeModal.tsx"
import {GAME_RULES} from "./BannedModal.tsx"
import {useWarning} from "../warning/warningStore.ts"

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
    useWarning.setState(useWarning.getInitialState(), true)
})

function noticed() {
    act(() => useWarning.getState().notice())
}

function acknowledging() {
    return {acknowledgeNotice: vi.fn().mockResolvedValue(undefined)}
}

const understand = () => screen.getByRole("button", {name: "I understand these rules"})

async function hold(ms = NOTICE_HOLD_MS) {
    fireEvent.pointerDown(understand())
    await act(() => vi.advanceTimersByTimeAsync(ms))
}

describe("NoticeModal", () => {
    it("says another account plays from the connection, and that one player plays one account", () => {
        render(<NoticeModal notices={acknowledging()}/>)
        noticed()

        expect(screen.getByRole("heading", {name: "One account per player"})).not.toBeNull()
        expect(screen.getByText("Another account is playing from your connection at the same time as you.")).not.toBeNull()
        expect(screen.getByText("Playing several accounts, a guest one included, is cheating. The anti-cheat watches for it, and pauses the accounts it catches.")).not.toBeNull()
        expect(screen.getByText("Someone else at home, at school or at work? You can both play: each on your own account, and only that one.")).not.toBeNull()
    })

    it("links to the play fair rules in a new tab", () => {
        render(<NoticeModal notices={acknowledging()}/>)
        noticed()

        const link = screen.getByRole("link", {name: "Read the game rules"})
        expect(link.getAttribute("href")).toBe(GAME_RULES)
        expect(link.getAttribute("target")).toBe("_blank")
    })

    it("shows nothing while the account is not noticed", () => {
        render(<NoticeModal notices={acknowledging()}/>)

        expect(screen.queryByText("One account per player")).toBeNull()
    })

    it("waits for the warning first, when the account has both", () => {
        render(<NoticeModal notices={acknowledging()}/>)
        noticed()
        act(() => useWarning.getState().warn())

        expect(screen.queryByRole("button", {name: "I understand these rules"})).toBeNull()

        act(() => useWarning.getState().dismiss())

        expect(understand()).not.toBeNull()
    })

    it("acknowledges the notice once the button is held three seconds, then closes", async () => {
        const notices = acknowledging()
        render(<NoticeModal notices={notices}/>)
        noticed()

        await hold()

        expect(notices.acknowledgeNotice).toHaveBeenCalledTimes(1)
        expect(screen.queryByText("One account per player")).toBeNull()
        expect(useWarning.getState().noticed).toBe(false)
    })

    it("takes no click, and no hold shorter than three seconds", async () => {
        const notices = acknowledging()
        render(<NoticeModal notices={notices}/>)
        noticed()

        fireEvent.click(understand())
        await hold(NOTICE_HOLD_MS - 100)
        fireEvent.pointerUp(understand())
        await act(() => vi.advanceTimersByTimeAsync(NOTICE_HOLD_MS))

        expect(notices.acknowledgeNotice).not.toHaveBeenCalled()
        expect(useWarning.getState().noticed).toBe(true)
    })

    it("cannot be closed any other way", () => {
        render(<NoticeModal notices={acknowledging()}/>)
        noticed()

        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()
        fireEvent.keyDown(document, {key: "Escape"})
        fireEvent.click(document.querySelector(".modal") as Element)

        expect(useWarning.getState().noticed).toBe(true)
        expect(screen.getByText("One account per player")).not.toBeNull()
    })

    it("stays open when the server could not take it, so the player can hold again", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const notices = {acknowledgeNotice: vi.fn().mockRejectedValueOnce(new Error("down")).mockResolvedValueOnce(undefined)}
        render(<NoticeModal notices={notices}/>)
        noticed()

        await hold()

        expect(logged).toHaveBeenCalledTimes(1)
        expect(useWarning.getState().noticed).toBe(true)

        fireEvent.pointerUp(understand())
        await hold()

        expect(notices.acknowledgeNotice).toHaveBeenCalledTimes(2)
        expect(useWarning.getState().noticed).toBe(false)
    })
})
