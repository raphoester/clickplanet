import {describe, expect, it} from "vitest"
import {anthemOf} from "./music.ts"
import {Story} from "./story.ts"

const recorded = (country: string) => ["fr", "nl", "tr", "es"].includes(country)

function story(attacker: string, place: Story["place"], rival?: string): Story {
    return {kind: "invasion", attacker, rival, victims: ["es", "fr"], place, taken: 100}
}

describe("the anthem of a clip", () => {
    it("is the leading flag's", () => {
        expect(anthemOf(story("fr", {country: "au"}), recorded)).toBe("fr")
    })

    it("is the other side's when the leading flag has no recording", () => {
        expect(anthemOf(story("ps", {country: "fr"}, "nl"), recorded)).toBe("nl")
    })

    it("is the place's when no side has one", () => {
        expect(anthemOf(story("ps", {country: "tr"}), recorded)).toBe("tr")
    })

    it("is a victim's when the place is a continent", () => {
        expect(anthemOf(story("ps", {region: "Europe"}), recorded)).toBe("es")
    })

    it("is nothing when no flag in the story has one", () => {
        expect(anthemOf(story("ps", {region: "Europe"}), () => false)).toBeUndefined()
    })
})
