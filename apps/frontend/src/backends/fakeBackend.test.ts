import {afterEach, describe, expect, it} from "vitest"
import {FakeBackend} from "./fakeBackend.ts"
import {MapFrozenError} from "./backend.ts"

let fake: FakeBackend | undefined

afterEach(() => fake?.close())

describe("FakeBackend.freeze", () => {
    it("refuses every write on the map as frozen until it thaws", async () => {
        fake = new FakeBackend(100)
        fake.grantBonus("refill")
        fake.grantBonus("shields")
        fake.grantBomb()

        fake.freeze()
        await expect(fake.clickTile(1, "fr")).rejects.toBeInstanceOf(MapFrozenError)
        await expect(fake.useRefill()).rejects.toBeInstanceOf(MapFrozenError)
        await expect(fake.placeShield(1, "fr")).rejects.toBeInstanceOf(MapFrozenError)
        await expect(fake.dropBomb({x: 1, y: 0, z: 0}, "fr")).rejects.toBeInstanceOf(MapFrozenError)
        await expect(fake.claimBonus("box", "fr")).rejects.toBeInstanceOf(MapFrozenError)
        await expect(fake.answerQuiz("quiz", 0, "fr")).rejects.toBeInstanceOf(MapFrozenError)

        fake.freeze(false)
        await expect(fake.clickTile(1, "fr")).resolves.toBeUndefined()
    })

    it("takes back the quiz it offered", async () => {
        fake = new FakeBackend(100)
        const offer = fake.offerQuiz()

        fake.freeze()
        fake.freeze(false)

        await expect(fake.openQuiz(offer.token)).rejects.toThrow()
    })
})
