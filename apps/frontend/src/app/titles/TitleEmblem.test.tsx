// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render} from "@testing-library/react"
import TitleEmblem from "./TitleEmblem.tsx"

afterEach(cleanup)

describe("TitleEmblem", () => {
    it("draws the first letter of a title this build has no picture for", () => {
        const {container} = render(<TitleEmblem title={{id: "grand_warmaster", name: "Grand Warmaster"}} size={56}/>)

        expect(container.querySelector("text")?.textContent).toBe("G")
    })

    it("gives each medal its own masks, so two on one page do not share one", () => {
        const {container} = render(<>
            <TitleEmblem title={{id: "settler", name: "Settler"}} size={56}/>
            <TitleEmblem title={{id: "raider", name: "Raider"}} size={56}/>
        </>)

        const ids = [...container.querySelectorAll("mask")].map((mask) => mask.id)
        expect(ids).toHaveLength(4)
        expect(new Set(ids).size).toBe(4)
        for (const masked of container.querySelectorAll("[mask]")) {
            expect(ids).toContain(masked.getAttribute("mask")?.slice("url(#".length, -1))
        }
    })

    it("draws the ribbon in the track's color only when asked", () => {
        const devoted = {id: "devoted", name: "Devoted", rank: {trackId: "devotion", trackName: "Devotion", number: 2, count: 3}}
        const plain = render(<TitleEmblem title={devoted} size={56}/>).container
        const ribboned = render(<TitleEmblem title={devoted} size={56} ribbon/>).container

        const ribbons = (container: HTMLElement) => [...container.querySelectorAll("path")]
            .filter((path) => path.style.fill === "var(--devotion)")
        expect(ribbons(plain)).toHaveLength(0)
        expect(ribbons(ribboned)).toHaveLength(2)
    })

    it("draws each track in its own colors, and a track this build does not know as Conquest", () => {
        const ranked = (id: string, trackId: string) => ({id, name: id, rank: {trackId, trackName: trackId, number: 1, count: 4}})
        const fills = (container: HTMLElement) => [...container.querySelectorAll("path, circle")]
            .map((shape) => (shape as SVGElement).style.fill)

        for (const track of ["defense", "bombs", "loot", "quiz", "missions"]) {
            const {container} = render(<TitleEmblem title={ranked(`${track}-rank`, track)} size={56} ribbon/>)
            expect(fills(container)).toContain(`var(--${track})`)
            expect(fills(container)).toContain(`var(--${track}-enamel)`)
        }
        expect(fills(render(<TitleEmblem title={ranked("x", "tomorrow")} size={56} ribbon/>).container)).toContain("var(--conquest)")
    })

    it("wreathes a season's title in the season's colors and writes its season on a scroll", () => {
        const {container} = render(<TitleEmblem title={{id: "season_0_silver", name: "Season 0 Silver"}} size={56} ribbon/>)

        expect([...container.querySelectorAll("text")].map((text) => text.textContent)).toEqual(["2", "S", "0"])
        const fills = [...container.querySelectorAll("path, circle")].map((shape) => (shape as SVGElement).style.fill)
        expect(fills).toContain("var(--season)")
        expect(fills).toContain("var(--season-enamel)")
    })

    it("draws a battle's top fighter as #2 and a season's top player as a star and 2, in the season's frame", () => {
        const texts = (id: string) => [...render(<TitleEmblem title={{id, name: id}} size={56} ribbon/>).container
            .querySelectorAll("text")].map((text) => text.textContent)

        expect(texts("season_0_battle_top_2")).toEqual(["#2", "S", "0"])
        expect(texts("season_0_top_2")).toEqual(["2", "S", "0"])
        const {container} = render(<TitleEmblem title={{id: "season_0_top_2", name: "Season 0 Top 2"}} size={56} ribbon/>)
        expect([...container.querySelectorAll("path, circle")].map((shape) => (shape as SVGElement).style.fill))
            .toContain("var(--season-enamel)")
    })

    it("greys a locked medal and hangs a padlock on it, with no ribbon", () => {
        const {container} = render(<TitleEmblem title={{id: "warlord", name: "Warlord"}} size={56} locked ribbon/>)

        expect(container.querySelector("svg")?.classList.contains("title-emblem-locked")).toBe(true)
        expect(container.querySelector("rect")).not.toBeNull()
        expect([...container.querySelectorAll("path")].some((path) => path.style.fill === "var(--og)")).toBe(false)
    })
})
