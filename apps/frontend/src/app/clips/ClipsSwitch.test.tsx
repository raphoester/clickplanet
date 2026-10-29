// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ClipsSwitch from "./ClipsSwitch.tsx"

afterEach(cleanup)

const clipsWith = (shown: boolean) => ({
    nameShown: vi.fn(async () => shown),
    setNameShown: vi.fn(async (on: boolean) => on),
})

const toggle = async () => await screen.findByRole("switch", {name: "Show my name in clips"}) as HTMLInputElement

describe("ClipsSwitch", () => {
    it("shows the choice the server keeps", async () => {
        render(<ClipsSwitch clips={clipsWith(false)}/>)

        expect((await toggle()).checked).toBe(false)
    })

    it("sends the choice and shows the one stored", async () => {
        const clips = clipsWith(true)
        render(<ClipsSwitch clips={clips}/>)

        await userEvent.click(await toggle())

        expect(clips.setNameShown).toHaveBeenCalledWith(false)
        expect((await toggle()).checked).toBe(false)
    })

    it("says a choice was refused, and shows the one kept", async () => {
        const clips = clipsWith(true)
        clips.setNameShown.mockRejectedValue(new Error("down"))
        render(<ClipsSwitch clips={clips}/>)

        await userEvent.click(await toggle())

        expect(await screen.findByRole("alert")).toBeDefined()
        expect((await toggle()).checked).toBe(true)
    })

    it("is not shown until the server says", () => {
        render(<ClipsSwitch clips={{nameShown: () => new Promise(() => undefined), setNameShown: vi.fn()}}/>)

        expect(screen.queryByRole("switch")).toBeNull()
    })
})
