// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import News, {FORTIFY_CARD_DELAY_MS} from "./News.tsx"
import {useNews} from "./newsStore.ts"
import {BombDrop} from "../../backends/backend.ts"
import {Fortified} from "../../domain/fortify.ts"
import {Moved} from "../../domain/missionProgress.ts"
import {Countries} from "../../domain/countries.ts"
import {useCountry} from "../countryStore.ts"

const DROP: BombDrop = {tile: 12, point: {x: 0, y: 0, z: 1}, countryId: "fr", radius: 0.03, cleared: [12, 13], struck: [], version: {epoch: 1, seq: 4}}
const CORSICA: Fortified = {fortification: {landmass: 257, countryId: "bg", tile: 107526, version: {epoch: 1, seq: 9}}, name: "Corsica", tiles: 16, news: true}
const PROGRESS: Moved = {id: "rival-3", order: {kind: "takeFrom", country: "ie", goal: 80, taken: 3}, flag: false}

function news(progress?: Moved) {
    useCountry.setState({country: Countries.get("fr")!})
    render(<News goThere={() => {}} progress={progress}/>)
}

describe("News", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
        useNews.setState(useNews.getInitialState(), true)
        useCountry.setState(useCountry.getInitialState(), true)
        window.localStorage.clear()
    })

    it("shows the mission's progress only while no news is told", () => {
        news(PROGRESS)
        expect(screen.getByRole("progressbar")).toBeTruthy()

        act(() => useNews.getState().tellBomb(DROP, undefined))
        expect(screen.queryByRole("progressbar")).toBeNull()
        expect(screen.getByRole("status").textContent).toContain("France")

        act(() => useNews.getState().tellAttack())
        expect(screen.getByRole("status").textContent).toContain("France is under attack")
    })

    it("brings the card on fortify after the wave, until this browser has read it once", () => {
        news()

        act(() => useNews.getState().tellFortified(CORSICA))
        act(() => void vi.advanceTimersByTime(FORTIFY_CARD_DELAY_MS - 1))
        expect(screen.queryByRole("button", {name: "Got it"})).toBeNull()

        act(() => void vi.advanceTimersByTime(1))
        fireEvent.click(screen.getByRole("button", {name: "Got it"}))
        expect(screen.queryByRole("button", {name: "Got it"})).toBeNull()

        act(() => useNews.getState().tellFortified({...CORSICA}))
        act(() => void vi.advanceTimersByTime(FORTIFY_CARD_DELAY_MS))
        expect(screen.queryByRole("button", {name: "Got it"})).toBeNull()
    })
})
