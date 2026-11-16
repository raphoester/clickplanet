// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {Ladder, NO_LADDER} from "../../domain/bonus.ts"
import {PLAIN_TURN} from "../../backends/turn.ts"
import {acceptedClicks} from "../clicks/acceptedClicks.ts"
import {SETTLE_MS} from "../clicks/useAcceptedClicks.ts"
import {NO_SCHEDULE, useSchedule} from "../battle/scheduleStore.ts"
import {useGatesFeed, useLadder} from "./ladderStore.ts"

const ladder = {level: 3, unlocks: {encloseClicks: 4, bomb: 5}}

const reader = (read = vi.fn(async (): Promise<Ladder | undefined> => ladder)) => ({ladder: read})
const refresher = () => ({refreshGrantRules: vi.fn(async () => {})})

function sessionsOf() {
    let tell: (token: string) => void = () => {}
    return {
        sessions: {listenForSessions: vi.fn((onSession: (token: string) => void) => {
            tell = onSession
            return () => {}
        })},
        tell: (token: string) => act(() => tell(token)),
    }
}

const feed = (ladder: ReturnType<typeof reader>, grantRules = refresher(), sessions = sessionsOf().sessions) =>
    ({ladder, grantRules, sessions})

const caller = (accountId: string | undefined, linked = false, listenForClicks = acceptedClicks().listenForClicks) =>
    ({accountId, linked, listenForClicks})

afterEach(() => {
    cleanup()
    useLadder.setState({ladder: NO_LADDER})
    useSchedule.setState(NO_SCHEDULE)
    vi.useRealTimers()
})

describe("useGatesFeed", () => {
    it("reads the ladder at load, and again when the account changes", async () => {
        const levels = reader()
        const sources = feed(levels)
        const {rerender} = renderHook(({account}) => useGatesFeed(sources, caller(account)),
            {initialProps: {account: "acc-1" as string | undefined}})
        await act(async () => {})
        expect(useLadder.getState().ladder).toEqual(ladder)

        rerender({account: undefined})
        expect(useLadder.getState().ladder).toEqual(NO_LADDER)
        await act(async () => {})
        expect(levels.ladder).toHaveBeenCalledTimes(2)
    })

    it("keeps no ladder while none can be read", async () => {
        renderHook(() => useGatesFeed(feed(reader(vi.fn(async (): Promise<Ladder | undefined> => undefined))), caller(undefined)))
        await act(async () => {})

        expect(useLadder.getState().ladder).toEqual(NO_LADDER)
    })

    it("reads it again once a signed-in player's clicks settle, a guest's never", async () => {
        vi.useFakeTimers()
        const clicks = acceptedClicks()
        const signedIn = reader()
        const guest = reader()
        renderHook(() => useGatesFeed(feed(signedIn), caller("acc-1", true, clicks.listenForClicks)))
        renderHook(() => useGatesFeed(feed(guest), caller(undefined, false, clicks.listenForClicks)))
        await act(async () => {})

        act(() => clicks.record({country: "fr", took: true}))
        await act(async () => vi.advanceTimersByTime(SETTLE_MS))

        expect(signedIn.ladder).toHaveBeenCalledTimes(2)
        expect(guest.ladder).toHaveBeenCalledTimes(1)
    })

    it("reads the bonuses' rules again when the turn moves to another entry, not on the first turn heard", async () => {
        const grantRules = refresher()
        renderHook(() => useGatesFeed(feed(reader(), grantRules), caller(undefined)))

        act(() => useSchedule.getState().hearTurn({...PLAIN_TURN, entryId: "day"}))
        act(() => useSchedule.getState().hearTurn({...PLAIN_TURN, entryId: "day", description: "same entry"}))
        expect(grantRules.refreshGrantRules).not.toHaveBeenCalled()

        act(() => useSchedule.getState().hearTurn({...PLAIN_TURN, kind: "battle", entryId: "battle"}))
        expect(grantRules.refreshGrantRules).toHaveBeenCalledOnce()
    })

    it("reads it again on a new session, as a guest's first click makes one", async () => {
        const levels = reader()
        const {sessions, tell} = sessionsOf()
        renderHook(() => useGatesFeed(feed(levels, refresher(), sessions), caller(undefined)))
        await act(async () => {})

        tell("guest-token")
        await act(async () => {})
        tell("guest-token")
        await act(async () => {})

        expect(levels.ladder).toHaveBeenCalledTimes(2)
    })
})
