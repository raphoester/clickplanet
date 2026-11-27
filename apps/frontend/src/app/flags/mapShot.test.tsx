// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, renderHook} from "@testing-library/react"
import type {Pin} from "../../domain/flags.ts"
import {FlagShooter, MapShot, ShotPicture, useMapShot} from "./mapShot.ts"

const MUNICH: Pin = {latitude: 48.14, longitude: 11.58}
const NUREMBERG: Pin = {latitude: 49.45, longitude: 11.08}
const SHOT: MapShot = {url: "blob:map", width: 560, height: 280}

const png = () => new File(["png"], "bavaria.png", {type: "image/png"})

function shotOf(shooter: FlagShooter, pin: Pin | undefined, picture: ShotPicture | undefined) {
    return renderHook(({pin, picture}) => useMapShot(shooter, pin, picture), {initialProps: {pin, picture}})
}

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("useMapShot", () => {
    it("shoots the map once a picture has a capital, and keeps the shot for the visit", async () => {
        const shooter = vi.fn(async () => SHOT)
        const file = png()
        const first = shotOf(shooter, MUNICH, file)
        await vi.waitFor(() => expect(first.result.current).toEqual(SHOT))
        first.unmount()

        const again = shotOf(shooter, MUNICH, file)
        expect(again.result.current, "the paywall shows the form's").toEqual(SHOT)
        expect(shooter).toHaveBeenCalledOnce()
    })

    it("shoots again for another capital, and keeps the one before", async () => {
        const shooter = vi.fn(async (pin: Pin) => ({...SHOT, url: `blob:${pin.latitude}`}))
        const file = png()
        const view = shotOf(shooter, MUNICH, file)
        await vi.waitFor(() => expect(view.result.current?.url).toBe("blob:48.14"))

        view.rerender({pin: NUREMBERG, picture: file})
        expect(view.result.current).toBeUndefined()
        await vi.waitFor(() => expect(view.result.current?.url).toBe("blob:49.45"))

        view.rerender({pin: MUNICH, picture: file})
        expect(view.result.current?.url).toBe("blob:48.14")
        expect(shooter).toHaveBeenCalledTimes(2)
    })

    it("shoots the public picture of an approved flag by its address", async () => {
        const shooter = vi.fn(async () => SHOT)
        const picture = {url: "https://media.clickplanet.lol/flags/pictures/cd.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}
        const view = shotOf(shooter, MUNICH, picture)
        await vi.waitFor(() => expect(view.result.current).toEqual(SHOT))

        shotOf(shooter, MUNICH, {...picture})
        expect(shooter).toHaveBeenCalledOnce()
    })

    it("shoots nothing with no picture or no capital", () => {
        const shooter = vi.fn(async () => SHOT)
        shotOf(shooter, undefined, png())
        shotOf(shooter, MUNICH, undefined)

        expect(shooter).not.toHaveBeenCalled()
    })

    it("keeps no shot that failed, and warns of it unless another shot took its place", async () => {
        const warn = vi.spyOn(console, "warn").mockImplementation(() => {})
        const view = shotOf(vi.fn(async () => Promise.reject(new DOMException("another preview was asked for", "AbortError"))), MUNICH, png())
        await new Promise((resolve) => setTimeout(resolve, 0))
        expect(view.result.current).toBeUndefined()
        expect(warn).not.toHaveBeenCalled()

        shotOf(vi.fn(async () => Promise.reject(new Error("the globe is not running yet"))), MUNICH, png())
        await vi.waitFor(() => expect(warn).toHaveBeenCalledOnce())
    })
})
