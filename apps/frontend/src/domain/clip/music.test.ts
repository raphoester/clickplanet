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

    it("is never a loser's: nothing when the flags making the moves have no recording", () => {
        expect(anthemOf(story("ps", {country: "tr"}), recorded)).toBeUndefined()
        expect(anthemOf(story("ps", {region: "Europe"}), recorded)).toBeUndefined()
    })
})
