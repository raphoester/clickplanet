// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {PlayerLevel} from "../../backends/progression.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {LevelReader, useLevel, useLevelFeed} from "./levelStore.ts"

const eleven: PlayerLevel = {level: 11, xp: 14_000, levelXp: 14_000, nextLevelXp: 18_634}
const twelve: PlayerLevel = {level: 12, xp: 18_700, levelXp: 18_634, nextLevelXp: 24_192}

type HarnessProps = {reader: LevelReader, accountId?: string, linked: boolean, listenForClicks?: ListenForClicks}

function Harness({reader, accountId = "account-1", linked, listenForClicks = () => () => {}}: HarnessProps) {
    useLevelFeed(reader, accountId, linked, listenForClicks)
    return null
}

const reading = (...levels: (PlayerLevel | undefined)[]) => {
    const level = vi.fn<() => Promise<PlayerLevel | undefined>>()
    for (const answer of levels) level.mockResolvedValueOnce(answer)
    return {level}
}

afterEach(() => {
    cleanup()
    useLevel.getState().forget()
})

describe("useLevelFeed", () => {
    it("reads a signed-in account's level into the store", async () => {
        const reader = reading(eleven)
        render(<Harness reader={reader} linked/>)

        await vi.waitFor(() => expect(useLevel.getState().own).toEqual(eleven))
    })

    it("reads nothing for a guest, which has no level", () => {
        const reader = reading(eleven)
        render(<Harness reader={reader} linked={false}/>)

        expect(reader.level).not.toHaveBeenCalled()
        expect(useLevel.getState().own).toBeUndefined()
    })

    it("forgets the level when the account signs out", async () => {
        const reader = reading(eleven)
        const view = render(<Harness reader={reader} linked/>)
        await vi.waitFor(() => expect(useLevel.getState().own).toEqual(eleven))

        view.rerender(<Harness reader={reader} linked={false}/>)

        expect(useLevel.getState().own).toBeUndefined()
    })

    it("keeps a level reached over a read that answers the one before", async () => {
        let answer: (level: PlayerLevel) => void = () => {}
        const reader = {level: vi.fn(() => new Promise<PlayerLevel | undefined>((resolve) => answer = resolve))}
        render(<Harness reader={reader} linked/>)

        act(() => useLevel.getState().heard(twelve))
        await act(async () => answer(eleven))

        expect(useLevel.getState().own).toEqual(twelve)
    })

    it("reads again once a run of clicks settles", async () => {
        vi.useFakeTimers()
        let click = () => {}
        const listenForClicks: ListenForClicks = (onClick) => {
            click = () => onClick({tile: 1, countryCode: "fr"} as never)
            return () => {}
        }
        const reader = reading(eleven, {...eleven, xp: 14_200})
        render(<Harness reader={reader} linked listenForClicks={listenForClicks}/>)

        act(() => click())
        await act(async () => vi.advanceTimersByTimeAsync(60_000))
        vi.useRealTimers()

        await vi.waitFor(() => expect(useLevel.getState().own).toEqual({...eleven, xp: 14_200}))
        expect(reader.level).toHaveBeenCalledTimes(2)
    })

    it("keeps the level read in StrictMode", async () => {
        const reader = {level: vi.fn(async () => eleven)}
        render(<StrictMode><Harness reader={reader} linked/></StrictMode>)

        await vi.waitFor(() => expect(useLevel.getState().own).toEqual(eleven))
    })
})
