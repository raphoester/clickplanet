import {describe, expect, it} from "vitest"
import {anthemOf} from "./music.ts"
import {Story} from "./story.ts"

const recorded = (country: string) => ["fr", "nl", "tr", "es", "eu"].includes(country)

function story(attacker: string, place: Story["place"], rival?: string): Story {
    return {kind: "invasion", attacker, rival, victims: ["es", "fr"], place, taken: 100}
}

describe("the anthem of a clip", () => {
    it("is the leading flag's", () => {
        expect(anthemOf(story("fr", {country: "au"}), recorded)).toBe("fr")
    })

    it("is the other side's in a battle when the leading flag has no recording", () => {
        expect(anthemOf(story("ps", {country: "fr"}, "nl"), recorded)).toBe("nl")
    })

    it("is the continent's when a continent strikes back together", () => {
        expect(anthemOf({...story("de", {region: "Europe"}), kind: "comeback", team: "Europe"}, recorded)).toBe("eu")
    })

    it("is the continent's when it throws a flag out", () => {
        expect(anthemOf({...story("de", {region: "Europe"}), kind: "rout", victims: ["ps"], team: "Europe"}, recorded)).toBe("eu")
    })

    it("is the fallen flag's when many flags throw it out and none leads, not the one that took the most", () => {
        expect(anthemOf({...story("tr", {country: "fr"}), kind: "rout", victims: ["fr"]}, recorded)).toBe("fr")
    })

    it("is never a loser's when a flag leads: nothing when the flags making the moves have no recording", () => {
        expect(anthemOf(story("ps", {country: "tr"}), recorded)).toBeUndefined()
        expect(anthemOf(story("ps", {region: "Europe"}), recorded)).toBeUndefined()
    })
})
