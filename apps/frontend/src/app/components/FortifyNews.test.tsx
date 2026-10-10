// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import FortifyNews, {FORTIFY_NEWS_MS} from "./FortifyNews.tsx"
import FortifyCard from "./FortifyCard.tsx"
import {Fortified} from "../../domain/fortify.ts"

const CORSICA: Fortified = {fortification: {landmass: 257, countryId: "bg", tile: 107526}, name: "Corsica", tiles: 16, news: true}

describe("FortifyNews", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("says which flag fortified which territory, and what each tile got", () => {
        render(<FortifyNews fortified={CORSICA} onDone={() => {}}/>)

        expect(screen.getByRole("status").textContent).toContain("Bulgaria fortified Corsica")
        expect(screen.getByText("+1 shield on 16 tiles")).toBeTruthy()
    })

    it("goes away on its own", () => {
        const onDone = vi.fn()
        render(<FortifyNews fortified={CORSICA} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(FORTIFY_NEWS_MS))

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})

describe("FortifyCard", () => {
    afterEach(cleanup)

    it("explains the rule on the territory that was just fortified, until the player has read it", () => {
        const onDone = vi.fn()
        render(<FortifyCard fortified={CORSICA} onDone={onDone}/>)

        expect(screen.getByText("Bulgaria owns all of Corsica, so every tile there gets +1 shield.")).toBeTruthy()
        expect(screen.getByText("A flag can't fortify the same territory twice in a row.")).toBeTruthy()

        fireEvent.click(screen.getByRole("button", {name: "Got it"}))

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
