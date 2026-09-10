// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render} from "@testing-library/react"
import CountryFlag from "./CountryFlag.tsx"
import {regions} from "../viewer/atlas.ts"
import {ATLAS_SIZE, ATLAS_URL} from "../viewer/atlasAsset.ts"
import {Countries} from "../../domain/countries.ts"

// The box the component draws in: one em wide, and as tall as the capitals it
// stands next to. Everything below is in em for the same reason the component
// is — a flag sized in px drifts as the text around it grows.
const BOX = {width: 1, height: 0.716}

afterEach(cleanup)

const flag = (container: HTMLElement) =>
    container.querySelector<HTMLElement>(".country-flag")
const sprite = (container: HTMLElement) =>
    container.querySelector<HTMLElement>(".country-flag-sprite")

const em = (value: string) => parseFloat(value)

describe("CountryFlag", () => {
    it("cuts the country's own region out of the atlas", () => {
        const {container} = render(<CountryFlag code="fr"/>)

        const fr = regions.get("fr")!
        const drawn = sprite(container)!.style
        const scale = em(drawn.width) / fr.width

        expect(drawn.backgroundImage).toBe(`url("${ATLAS_URL}")`)
        expect(drawn.backgroundSize)
            .toBe(`${ATLAS_SIZE.width * scale}em ${ATLAS_SIZE.height * scale}em`)
        expect(drawn.backgroundPosition)
            .toBe(`${-fr.x * scale}em ${-fr.y * scale}em`)
    })

    it("stands in a box the height of the capitals beside it", () => {
        const {container} = render(<CountryFlag code="fr"/>)

        expect(flag(container)!.style.width).toBe(`${BOX.width}em`)
        expect(flag(container)!.style.height).toBe(`${BOX.height}em`)
    })

    it("fits every flag inside that box, whatever its shape", () => {
        const drawn = Array.from(Countries.keys()).map(code => {
            const {container} = render(<CountryFlag code={code}/>)
            const style = sprite(container)!.style
            cleanup()
            return {code, width: em(style.width), height: em(style.height)}
        })

        const spilling = drawn.filter(d =>
            d.width > BOX.width + 1e-9 || d.height > BOX.height + 1e-9)
        expect(spilling).toEqual([])
        expect(drawn.every(d => d.width > 0 && d.height > 0)).toBe(true)
    })

    it("keeps a country's proportions", () => {
        const {container} = render(<CountryFlag code="np"/>)

        const np = regions.get("np")!
        const drawn = sprite(container)!.style
        expect(em(drawn.width) / em(drawn.height)).toBeCloseTo(np.width / np.height, 5)
    })

    it("centres a short flag with a margin, not with the box", () => {
        // The box has to keep its own bottom edge as its baseline: a box that
        // centres its content takes the content's baseline instead, and a column
        // of flags of different heights ends up jittering against the names.
        const {container} = render(<CountryFlag code="ne"/>)

        const drawn = sprite(container)!.style
        expect(em(drawn.marginTop)).toBeCloseTo((BOX.height - em(drawn.height)) / 2, 5)
    })

    it("draws nothing for a code the atlas does not cover", () => {
        const {container} = render(<CountryFlag code="zz"/>)
        expect(flag(container)).toBeNull()
    })

    it("leaves itself out of the accessibility tree, where the name already is", () => {
        const {container} = render(<CountryFlag code="fr"/>)
        expect(flag(container)!.getAttribute("aria-hidden")).toBe("true")
    })
})
