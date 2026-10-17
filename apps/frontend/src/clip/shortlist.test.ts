import {describe, expect, it} from "vitest"
import {Story} from "../domain/clip/story.ts"
import {shortlistOf} from "./shortlist.ts"

describe("the shortlist", () => {
    it("gives each story its headline, its key and every country it shows, best first", () => {
        const france: Story = {kind: "battle", attacker: "fr", rival: "pw", victims: ["pw"], place: {country: "fr"}, taken: 900}
        const vatican: Story = {kind: "battle", attacker: "va", rival: "pw", victims: ["pw"], place: {country: "va"}, taken: 400}

        expect(shortlistOf([france, vatican]).toJson()).toEqual({
            stories: [
                {title: "FRANCE VS PALAU", key: {flag: "fr", continent: "Europe"}, countries: ["fr", "pw"]},
                {title: "VATICAN CITY VS PALAU", key: {flag: "va", continent: "Europe"}, countries: ["va", "pw"]},
            ],
        })
    })
})
