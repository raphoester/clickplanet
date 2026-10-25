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

    it("draws a season's veteran and champion in the season's colors, whatever the season", () => {
        const fills = (container: HTMLElement) => [...container.querySelectorAll("path, circle")]
            .map((shape) => (shape as SVGElement).style.fill)

        for (const title of [{id: "veteran_s0", name: "Veteran · S0"}, {id: "champion_s3", name: "Champion · S3"}]) {
            const {container} = render(<TitleEmblem title={title} size={56} ribbon/>)
            expect(container.querySelector("text"), title.id).toBeNull()
            expect(fills(container)).toContain("var(--season)")
            expect(fills(container)).toContain("var(--season-enamel)")
        }
        expect(fills(render(<TitleEmblem title={{id: "champion_s0", name: "Champion · S0"}} size={56}/>).container))
            .toContain("var(--gold)")
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

        for (const track of ["defense", "bombs", "loot", "quiz"]) {
            const {container} = render(<TitleEmblem title={ranked(`${track}-rank`, track)} size={56} ribbon/>)
            expect(fills(container)).toContain(`var(--${track})`)
            expect(fills(container)).toContain(`var(--${track}-enamel)`)
        }
        expect(fills(render(<TitleEmblem title={ranked("x", "tomorrow")} size={56} ribbon/>).container)).toContain("var(--conquest)")
    })

    it("greys a locked medal and hangs a padlock on it, with no ribbon", () => {
        const {container} = render(<TitleEmblem title={{id: "warlord", name: "Warlord"}} size={56} locked ribbon/>)

        expect(container.querySelector("svg")?.classList.contains("title-emblem-locked")).toBe(true)
        expect(container.querySelector("rect")).not.toBeNull()
        expect([...container.querySelectorAll("path")].some((path) => path.style.fill === "var(--og)")).toBe(false)
    })
})
