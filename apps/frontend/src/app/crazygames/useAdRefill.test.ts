// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, renderHook, waitFor} from "@testing-library/react"
import {AdRefiller, AdRefillTooSoonError, BankFullError} from "../../backends/backend.ts"
import {AdOutcome, RewardedAds} from "../../backends/crazyGames.ts"
import {useAdRefill} from "./useAdRefill.ts"

afterEach(() => {
    vi.useRealTimers()
})

function ads(outcome: AdOutcome = "watched", blocked = false) {
    return {
        blocked: vi.fn(async () => blocked),
        watch: vi.fn(async (onStart: () => void) => {
            if (outcome === "watched") onStart()
            return outcome
        }),
    } satisfies RewardedAds
}

function refiller(availableAt = 0) {
    return {
        adRefillAt: vi.fn(async () => availableAt),
        claimAdRefill: vi.fn(async () => undefined),
    } satisfies AdRefiller
}

function adRefill(from: {ads?: RewardedAds, refiller?: AdRefiller, on?: boolean}) {
    const onPlaying = vi.fn()
    const hook = renderHook(() => useAdRefill(from.ads, from.refiller, from.on ?? true, onPlaying))
    return {...hook, onPlaying}
}

describe("useAdRefill", () => {
    it("is off without CrazyGames' ads, without the planet's refill, or with the rule off", async () => {
        for (const from of [{refiller: refiller()}, {ads: ads()}, {ads: ads(), refiller: refiller(), on: false}]) {
            const {result} = adRefill(from)
            await act(async () => undefined)
            expect(result.current.state).toEqual({kind: "off"})
        }
    })

    it("is ready when this bank has not had its ad refill", async () => {
        const {result} = adRefill({ads: ads(), refiller: refiller(0)})

        await waitFor(() => expect(result.current.state).toEqual({kind: "ready"}))
    })

    it("says when the next one is, and is ready then", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const until = Date.now() + 60_000
        const {result} = adRefill({ads: ads(), refiller: refiller(until)})
        await waitFor(() => expect(result.current.state).toEqual({kind: "waiting", until}))

        await act(async () => {
            vi.advanceTimersByTime(60_000)
        })

        expect(result.current.state).toEqual({kind: "ready"})
    })

    it("tells an adblocker apart, and never asks the server then", async () => {
        const planet = refiller()
        const {result} = adRefill({ads: ads("watched", true), refiller: planet})

        await waitFor(() => expect(result.current.state).toEqual({kind: "blocked"}))
        expect(planet.adRefillAt).not.toHaveBeenCalled()
    })

    it("asks the server for the refill once the ad played to its end, and mutes the game while it plays", async () => {
        const planet = refiller()
        const {result, onPlaying} = adRefill({ads: ads("watched"), refiller: planet})
        await waitFor(() => expect(result.current.state.kind).toBe("ready"))

        act(() => result.current.watch("fr"))

        await waitFor(() => expect(result.current.state).toEqual({kind: "refilled"}))
        expect(planet.claimAdRefill).toHaveBeenCalledWith("fr")
        expect(onPlaying.mock.calls).toEqual([[true], [false]])
    })

    it("never asks for the refill when the ad did not play", async () => {
        for (const [outcome, state] of [["unavailable", "unavailable"], ["blocked", "blocked"], ["off", "off"]] as const) {
            const planet = refiller()
            const {result} = adRefill({ads: ads(outcome), refiller: planet})
            await waitFor(() => expect(result.current.state.kind).toBe("ready"))

            act(() => result.current.watch("fr"))

            await waitFor(() => expect(result.current.state.kind).toBe(state))
            expect(planet.claimAdRefill).not.toHaveBeenCalled()
        }
    })

    it("waits as the server says when the bank had its ad refill", async () => {
        const planet = refiller()
        planet.claimAdRefill.mockRejectedValue(new AdRefillTooSoonError(Date.now() + 3_600_000))
        const {result} = adRefill({ads: ads("watched"), refiller: planet})
        await waitFor(() => expect(result.current.state.kind).toBe("ready"))

        act(() => result.current.watch("fr"))

        await waitFor(() => expect(result.current.state.kind).toBe("waiting"))
    })

    it("reads the clock again after a full bank refused it", async () => {
        const planet = refiller()
        planet.claimAdRefill.mockRejectedValue(new BankFullError())
        const {result} = adRefill({ads: ads("watched"), refiller: planet})
        await waitFor(() => expect(result.current.state.kind).toBe("ready"))

        act(() => result.current.watch("fr"))

        await waitFor(() => expect(planet.adRefillAt).toHaveBeenCalledTimes(2))
        expect(result.current.state).toEqual({kind: "ready"})
    })

    it("shows the clock to the next one once the refill was seen", async () => {
        const planet = refiller()
        const {result} = adRefill({ads: ads("watched"), refiller: planet})
        await waitFor(() => expect(result.current.state.kind).toBe("ready"))
        act(() => result.current.watch("fr"))
        await waitFor(() => expect(result.current.state.kind).toBe("refilled"))
        const until = Date.now() + 3_600_000
        planet.adRefillAt.mockResolvedValue(until)

        act(() => result.current.seen())

        await waitFor(() => expect(result.current.state).toEqual({kind: "waiting", until}))
    })
})
