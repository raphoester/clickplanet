// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {LevelRead, LevelReader, PlayerLevel} from "../../backends/progression.ts"
import {acceptedClicks, ListenForClicks} from "../clicks/acceptedClicks.ts"
import {SETTLE_MS} from "../clicks/useAcceptedClicks.ts"
import {HOLD_MS, TILES_HOLD_MS} from "../../domain/xpPops.ts"
import {useLevel, useLevelFeed} from "./levelStore.ts"

const eleven: PlayerLevel = {level: 11, xp: 14_000, levelXp: 14_000, nextLevelXp: 18_634}
const twelve: PlayerLevel = {level: 12, xp: 18_700, levelXp: 18_634, nextLevelXp: 24_192}
const unlocks = {encloseClicks: 4, bomb: 5}
const missionRates = {takeFrom: {xp: 3, per: 2}}
const xpRates = {tilesTaken: {xp: 1, per: 1}, tilesFortified: {xp: 1, per: 5}}

const answering = (...reads: (LevelRead | undefined)[]) => {
    const readLevel = vi.fn<() => Promise<LevelRead | undefined>>(async () => reads[reads.length - 1])
    for (const read of reads) readLevel.mockResolvedValueOnce(read)
    return {readLevel}
}

function sessionsOf() {
    let tell: (token: string) => void = () => {}
    return {
        sessions: {listenForSessions: (onSession: (token: string) => void) => {
            tell = onSession
            return () => {}
        }},
        tell: (token: string) => act(() => tell(token)),
    }
}

type HarnessProps = {reader: LevelReader, accountId?: string, linked: boolean, listenForClicks?: ListenForClicks, sessions?: ReturnType<typeof sessionsOf>["sessions"]}

function Harness({reader, accountId = "account-1", linked, listenForClicks = acceptedClicks().listenForClicks, sessions = sessionsOf().sessions}: HarnessProps) {
    useLevelFeed(reader, sessions, {accountId, linked, listenForClicks})
    return null
}

afterEach(() => {
    cleanup()
    useLevel.getState().forget()
    vi.useRealTimers()
})

describe("useLevelFeed", () => {
    it("reads the level and the ladder in one call", async () => {
        const reader = answering({level: eleven, unlocks, missionRates, xpRates})
        render(<Harness reader={reader} linked/>)

        await vi.waitFor(() => expect(useLevel.getState().own).toEqual(eleven))
        expect(useLevel.getState().unlocks).toEqual(unlocks)
        expect(useLevel.getState().missionRates).toEqual(missionRates)
        expect(useLevel.getState().xpRates).toEqual(xpRates)
        expect(reader.readLevel).toHaveBeenCalledTimes(1)
    })

    it("reads a guest's ladder, which has no level", async () => {
        render(<Harness reader={answering({unlocks, missionRates, xpRates})} linked={false}/>)

        await vi.waitFor(() => expect(useLevel.getState().unlocks).toEqual(unlocks))
        expect(useLevel.getState().own).toBeUndefined()
    })

    it("keeps nothing while no session is held to read with", async () => {
        render(<Harness reader={answering(undefined)} linked={false}/>)
        await act(async () => {})

        expect(useLevel.getState()).toMatchObject({own: undefined, unlocks: {}, missionRates: {}, xpRates: {}})
    })

    it("forgets the level and the ladder when the account signs out or changes, and reads again", async () => {
        const reader = answering({level: eleven, unlocks, missionRates, xpRates}, {unlocks, missionRates, xpRates})
        const view = render(<Harness reader={reader} linked/>)
        await vi.waitFor(() => expect(useLevel.getState().own).toEqual(eleven))

        view.rerender(<Harness reader={reader} accountId="account-2" linked={false}/>)

        expect(useLevel.getState().own).toBeUndefined()
        await vi.waitFor(() => expect(reader.readLevel).toHaveBeenCalledTimes(2))
    })

    it("keeps a level reached over a read that answers the one before", async () => {
        let answer: (read: LevelRead) => void = () => {}
        const reader = {readLevel: vi.fn(() => new Promise<LevelRead | undefined>((resolve) => answer = resolve))}
        render(<Harness reader={reader} linked/>)

        act(() => useLevel.getState().reached(twelve))
        await act(async () => answer({level: eleven, unlocks, missionRates, xpRates}))

        expect(useLevel.getState().own).toEqual(twelve)
        expect(useLevel.getState().unlocks).toEqual(unlocks)
    })

    it("reads again once a signed-in player's clicks settle, a guest's never", async () => {
        vi.useFakeTimers()
        const clicks = acceptedClicks()
        const signedIn = answering({level: eleven, unlocks, missionRates, xpRates})
        const guest = answering({unlocks, missionRates, xpRates})
        render(<Harness reader={signedIn} linked listenForClicks={clicks.listenForClicks}/>)
        render(<Harness reader={guest} accountId="guest-1" linked={false} listenForClicks={clicks.listenForClicks}/>)
        await act(async () => {})

        act(() => clicks.record({country: "fr", took: true}))
        await act(async () => vi.advanceTimersByTime(SETTLE_MS))

        expect(signedIn.readLevel).toHaveBeenCalledTimes(2)
        expect(guest.readLevel).toHaveBeenCalledTimes(1)
    })

    it("reads again on a new session, as a guest's first click makes one", async () => {
        const reader = answering({unlocks, missionRates, xpRates})
        const {sessions, tell} = sessionsOf()
        render(<Harness reader={reader} linked={false} sessions={sessions}/>)
        await act(async () => {})

        tell("guest-token")
        await act(async () => {})
        tell("guest-token")
        await act(async () => {})

        expect(reader.readLevel).toHaveBeenCalledTimes(2)
    })

    it("keeps the level read in StrictMode", async () => {
        render(<StrictMode><Harness reader={answering({level: eleven, unlocks, missionRates, xpRates})} linked/></StrictMode>)

        await vi.waitFor(() => expect(useLevel.getState().own).toEqual(eleven))
    })
})

describe("gained", () => {
    it("lets each gain go after its hold, then shows the next", () => {
        vi.useFakeTimers()
        act(() => useLevel.getState().gained({gains: [{source: "tilesTaken", xp: 1}, {source: "daysPlayed", xp: 100}]}))

        act(() => vi.advanceTimersByTime(HOLD_MS))
        expect(useLevel.getState().pops.shown?.source).toBe("tilesTaken")

        act(() => vi.advanceTimersByTime(TILES_HOLD_MS))
        expect(useLevel.getState().pops.shown).toBeUndefined()
    })

    it("keeps a run of takes on the line while they keep coming", () => {
        vi.useFakeTimers()
        for (let i = 0; i < 5; i++) {
            act(() => useLevel.getState().gained({gains: [{source: "tilesTaken", xp: 1}]}))
            act(() => vi.advanceTimersByTime(TILES_HOLD_MS - 100))
        }

        expect(useLevel.getState().pops.shown).toMatchObject({source: "tilesTaken", xp: 5})
    })

    it("forgets the gains with the level", () => {
        vi.useFakeTimers()
        act(() => useLevel.getState().gained({gains: [{source: "boxesCaught", xp: 30}]}))

        act(() => useLevel.getState().forget())

        expect(useLevel.getState().pops.shown).toBeUndefined()
    })
})
