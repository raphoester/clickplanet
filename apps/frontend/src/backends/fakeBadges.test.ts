import {describe, expect, it} from "vitest"
import {Badge, MAX_BADGES_ASKED} from "./badges.ts"
import {FakeBadges} from "./fakeBadges.ts"

const OG = {id: "og", name: "OG"}

const knowing = (badges: Record<string, Badge>) => ({badgeOf: (accountId: string) => badges[accountId]})

describe("FakeBadges", () => {
    it("answers each account from the first fake that knows it, once, and leaves out one with nothing to show", async () => {
        const roster = knowing({ana: {streak: 12, wornTitle: OG}, guest: {streak: 0}})
        const board = knowing({ana: {streak: 0}, kofi: {streak: 0, wornTitle: OG}})

        expect(await new FakeBadges([roster, board]).badges(["ana", "guest", "kofi", "nobody", "ana"])).toEqual([
            {accountId: "ana", streak: 12, wornTitle: OG},
            {accountId: "kofi", streak: 0, wornTitle: OG},
        ])
    })

    it("refuses more accounts than the server takes", async () => {
        const ids = Array.from({length: MAX_BADGES_ASKED + 1}, (_, i) => `account-${i}`)

        await expect(new FakeBadges([]).badges(ids)).rejects.toThrow()
    })
})
