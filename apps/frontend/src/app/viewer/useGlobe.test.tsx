// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, waitFor} from "@testing-library/react"
import {createRef, useRef} from "react"
import {useGlobe} from "./useGlobe.ts"
import type {Globe} from "./globe.ts"
import {Countries} from "../../domain/countries.ts"
import type {OwnershipsGetter, TileClicker, UpdatesListener} from "../../backends/backend.ts"

const createGlobe = vi.hoisted(() => vi.fn())
vi.mock("./globe.ts", () => ({createGlobe}))

const FRANCE = Countries.get("fr")!
const JAPAN = Countries.get("jp")!

const backends = () => ({
    tileClicker: {} as TileClicker,
    ownershipsGetter: {} as OwnershipsGetter,
    updatesListener: {} as UpdatesListener,
})

function fakeGlobe(): Globe & {
    setCountry: ReturnType<typeof vi.fn>,
    capture: ReturnType<typeof vi.fn>,
    dispose: ReturnType<typeof vi.fn>,
} {
    return {tilesCount: 257_948, setCountry: vi.fn(), takeReward: vi.fn(), capture: vi.fn(), dispose: vi.fn()}
}

function Harness(props: {country: typeof FRANCE, backends: ReturnType<typeof backends>, onResult: (r: unknown) => void}) {
    const container = useRef<HTMLDivElement>(null)
    props.onResult(useGlobe({container, ...props.backends, country: props.country}))
    return <div ref={container}/>
}

let latest: ReturnType<typeof useGlobe>

function renderHook(country = FRANCE, deps = backends()) {
    const view = render(<Harness country={country} backends={deps} onResult={r => {
        latest = r as ReturnType<typeof useGlobe>
    }}/>)
    return {
        ...view,
        rerenderWith: (next: typeof FRANCE, nextDeps = deps) =>
            view.rerender(<Harness country={next} backends={nextDeps} onResult={r => {
                latest = r as ReturnType<typeof useGlobe>
            }}/>),
    }
}

beforeEach(() => {
    createGlobe.mockReset()
    vi.spyOn(console, "error").mockImplementation(() => {})
})

afterEach(cleanup)

describe("useGlobe", () => {
    it("reports loading until the globe resolves", async () => {
        let resolve: (globe: Globe) => void = () => {}
        createGlobe.mockReturnValue(new Promise<Globe>(r => {resolve = r}))

        renderHook()
        expect(latest.status).toEqual({state: 'loading'})

        await act(async () => resolve(fakeGlobe()))
        expect(latest.status).toEqual({state: 'ready'})
        expect(latest.tilesCount).toBe(257_948)
    })

    it("reports the failure when the globe cannot be built", async () => {
        createGlobe.mockRejectedValue(new Error("WebGL unavailable"))

        renderHook()
        await waitFor(() => expect(latest.status).toEqual({state: 'failed', message: "WebGL unavailable"}))
    })

    it("builds the globe with the country selected at mount", async () => {
        createGlobe.mockResolvedValue(fakeGlobe())

        renderHook(JAPAN)
        await waitFor(() => expect(latest.status.state).toBe('ready'))
        expect(createGlobe.mock.calls[0][0]).toMatchObject({country: JAPAN})
    })

    it("does not rebuild the globe when the component re-renders", async () => {
        const globe = fakeGlobe()
        createGlobe.mockResolvedValue(globe)

        const {rerenderWith} = renderHook(FRANCE)
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        rerenderWith(FRANCE)
        rerenderWith(FRANCE)

        expect(createGlobe).toHaveBeenCalledTimes(1)
        expect(globe.dispose).not.toHaveBeenCalled()
    })

    it("pushes a country change into the running globe instead of rebuilding it", async () => {
        const globe = fakeGlobe()
        createGlobe.mockResolvedValue(globe)

        const {rerenderWith} = renderHook(FRANCE)
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await act(async () => {rerenderWith(JAPAN)})

        expect(globe.setCountry).toHaveBeenCalledWith(JAPAN)
        expect(createGlobe).toHaveBeenCalledTimes(1)
        expect(globe.dispose).not.toHaveBeenCalled()
    })

    it("rebuilds the globe when a backend is swapped", async () => {
        const globe = fakeGlobe()
        createGlobe.mockResolvedValue(globe)

        const {rerenderWith} = renderHook(FRANCE)
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await act(async () => {rerenderWith(FRANCE, backends())})

        expect(createGlobe).toHaveBeenCalledTimes(2)
        expect(globe.dispose).toHaveBeenCalledTimes(1)
    })

    it("disposes the globe on unmount", async () => {
        const globe = fakeGlobe()
        createGlobe.mockResolvedValue(globe)

        const {unmount} = renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        unmount()
        expect(globe.dispose).toHaveBeenCalledTimes(1)
    })

    it("disposes a globe that resolves after the component is gone", async () => {
        const globe = fakeGlobe()
        let resolve: (globe: Globe) => void = () => {}
        createGlobe.mockReturnValue(new Promise<Globe>(r => {resolve = r}))

        const {unmount} = renderHook()
        unmount()

        await act(async () => resolve(globe))
        expect(globe.dispose).toHaveBeenCalledTimes(1)
    })

    it("aborts the load when the component goes away first", async () => {
        createGlobe.mockReturnValue(new Promise(() => {}))

        const {unmount} = renderHook()
        const {signal} = createGlobe.mock.calls[0][0]
        expect(signal.aborted).toBe(false)

        unmount()
        expect(signal.aborted).toBe(true)
    })

    it("publishes leaderboard updates on the next sample", async () => {
        createGlobe.mockResolvedValue(fakeGlobe())

        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        const entries = [{country: FRANCE, tiles: 12}]
        await act(async () => createGlobe.mock.calls[0][0].onLeaderboardChange(entries, true))
        await waitFor(() => expect(latest.leaderboard).toEqual(entries))
    })

    it("badges what lands live, and not the map the globe starts from", async () => {
        createGlobe.mockResolvedValue(fakeGlobe())

        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        const change = createGlobe.mock.calls[0][0].onLeaderboardChange

        await act(async () => change([{country: FRANCE, tiles: 12}], false))
        await waitFor(() => expect(latest.leaderboard).toHaveLength(1))
        expect(latest.tileDeltas.size).toBe(0)

        await act(async () => change([{country: FRANCE, tiles: 14}], true))
        await waitFor(() => expect(latest.tileDeltas.get("fr")?.net).toBe(2))
    })

    it("hands a capture to the globe that is running", async () => {
        const globe = fakeGlobe()
        const frame = {pixels: new Uint8ClampedArray(4), width: 1, height: 1}
        globe.capture.mockResolvedValue(frame)
        createGlobe.mockResolvedValue(globe)

        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await expect(latest.capture()).resolves.toBe(frame)
    })

    // The share button is only mounted once the globe is ready, so this is the
    // fallback rather than the usual case — and a promise that never settles
    // would leave the button saying it was working for the rest of the session.
    it("refuses a capture before there is a globe, rather than waiting forever", async () => {
        createGlobe.mockReturnValue(new Promise(() => {}))

        renderHook()

        await expect(latest.capture()).rejects.toThrow()
    })

    it("does nothing until the container element exists", () => {
        createGlobe.mockResolvedValue(fakeGlobe())

        function NoContainer() {
            useGlobe({container: createRef<HTMLDivElement>(), ...backends(), country: FRANCE})
            return null
        }
        render(<NoContainer/>)

        expect(createGlobe).not.toHaveBeenCalled()
    })
})

describe("useGlobe rate limiting", () => {
    const refuseAClick = async () =>
        act(async () => createGlobe.mock.calls[0][0].onRateLimited())

    beforeEach(() => createGlobe.mockResolvedValue(fakeGlobe()))

    it("stays quiet until the server refuses a click", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        expect(latest.rateLimited).toBe(false)
    })

    it("raises the flag when the globe reports a refused click", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await refuseAClick()
        expect(latest.rateLimited).toBe(true)
    })

    it("stays raised across a burst of refusals", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await refuseAClick()
        await refuseAClick()
        await refuseAClick()
        expect(latest.rateLimited).toBe(true)
    })

    it("lowers the flag when the player dismisses it, and raises it again after", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await refuseAClick()
        await act(async () => latest.dismissRateLimited())
        expect(latest.rateLimited).toBe(false)

        await refuseAClick()
        expect(latest.rateLimited).toBe(true)
    })
})

describe("useGlobe VPN blocking", () => {
    const blockAClick = async () =>
        act(async () => createGlobe.mock.calls[0][0].onVPNBlocked())

    beforeEach(() => createGlobe.mockResolvedValue(fakeGlobe()))

    it("stays quiet until the server refuses a click", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        expect(latest.vpnBlocked).toBe(false)
    })

    it("raises the flag when the globe reports a blocked click", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await blockAClick()
        expect(latest.vpnBlocked).toBe(true)
    })

    it("lowers the flag when the player dismisses it, and raises it again after", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await blockAClick()
        await act(async () => latest.dismissVPNBlocked())
        expect(latest.vpnBlocked).toBe(false)

        await blockAClick()
        expect(latest.vpnBlocked).toBe(true)
    })

    it("does not touch the throttle's flag, and is not touched by it", async () => {
        renderHook()
        await waitFor(() => expect(latest.status.state).toBe('ready'))

        await blockAClick()
        expect(latest.rateLimited).toBe(false)

        await act(async () => createGlobe.mock.calls[0][0].onRateLimited())
        await act(async () => latest.dismissVPNBlocked())
        expect(latest.rateLimited).toBe(true)
    })
})
