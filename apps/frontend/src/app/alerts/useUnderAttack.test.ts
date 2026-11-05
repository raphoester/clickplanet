// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import {act, renderHook} from "@testing-library/react"
import {StrictMode} from "react"
import {NO_TILE_DELTAS, TileDeltas} from "../../domain/tileDeltas.ts"
import {acceptedClicks} from "../viewer/acceptedClicks.ts"
import {useUnderAttack} from "./useUnderAttack.ts"

type Props = {flag: string, deltas: TileDeltas}

const losing = (code: string, net = -2): TileDeltas => new Map([[code, {net, beat: 1, at: 0}]])

function underAttack(initialProps: Props = {flag: "fr", deltas: NO_TILE_DELTAS}) {
    const clicks = acceptedClicks()
    const hook = renderHook((props: Props) => useUnderAttack(props.flag, props.deltas, clicks.listenForClicks),
        {initialProps, wrapper: StrictMode})
    const take = (country: string) => act(() => clicks.record({country, took: true}))
    return {...hook, take}
}

describe("useUnderAttack", () => {
    it("shows when the flag the player took a tile for loses ground", () => {
        const {result, rerender, take} = underAttack()

        take("fr")
        rerender({flag: "fr", deltas: losing("fr")})

        expect(result.current.moment).toBeDefined()
    })

    it("waits for a take before a loss counts", () => {
        const {result, rerender, take} = underAttack()

        rerender({flag: "fr", deltas: losing("fr")})
        expect(result.current.moment).toBeUndefined()

        take("fr")
        expect(result.current.moment).toBeDefined()
    })

    it("does not count a take for another flag, a gain, or another flag's loss", () => {
        const {result, rerender, take} = underAttack()

        take("de")
        rerender({flag: "fr", deltas: losing("fr")})
        rerender({flag: "fr", deltas: losing("fr", 3)})
        take("fr")
        rerender({flag: "fr", deltas: losing("de")})

        expect(result.current.moment).toBeUndefined()
    })

    it("shows once a page, whatever is lost after", () => {
        const {result, rerender, take} = underAttack()

        take("fr")
        rerender({flag: "fr", deltas: losing("fr")})
        act(() => result.current.dismiss())
        rerender({flag: "fr", deltas: losing("fr", -5)})

        expect(result.current.moment).toBeUndefined()
    })
})
