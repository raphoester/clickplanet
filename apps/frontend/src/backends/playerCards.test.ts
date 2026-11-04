import {describe, expect, it, vi} from "vitest"
import {NameColor, Player} from "./player.ts"
import {PlayerCards} from "./playerCards.ts"
import {Progress} from "./progression.ts"

const ana: Player = {name: "Ana", guest: false, createdAt: 1_788_000_000_000, admin: true, color: NameColor.VIOLET}
const og = {id: "og", name: "OG"}
const anaProgress: Progress = {
    tilesTaken: 1234, streakCurrent: 3, streakBest: 7, titles: [og], wornTitle: og,
    playsFor: [{countryCode: "fr", tiles: 1200}], playsAgainst: [{countryCode: "de", tiles: 900}],
}
const zeros = {tilesTaken: 0, streakCurrent: 0, streakBest: 0, titles: [], playsFor: [], playsAgainst: []}

function cardsOf(player: () => Promise<Player | undefined>, progress: () => Promise<Progress | undefined>) {
    const players = {player: vi.fn(player)}
    const progression = {progress: vi.fn(progress)}
    return {cards: new PlayerCards(players, progression), players, progression}
}

describe("PlayerCards", () => {
    it("shows who the player is and how far it went, both read by its account id", async () => {
        const {cards, players, progression} = cardsOf(async () => ana, async () => anaProgress)

        expect(await cards.playerInfo("account-ana")).toEqual({...ana, ...anaProgress})
        expect(players.player).toHaveBeenCalledWith("account-ana")
        expect(progression.progress).toHaveBeenCalledWith("account-ana")
    })

    it("reads both at once", () => {
        const {cards, players, progression} = cardsOf(() => new Promise(() => {}), () => new Promise(() => {}))

        void cards.playerInfo("account-ana")

        expect(players.player).toHaveBeenCalledTimes(1)
        expect(progression.progress).toHaveBeenCalledTimes(1)
    })

    it("shows a guest as its name alone", async () => {
        const guest: Player = {name: "guest_91aa3d", guest: true, admin: false, color: NameColor.UNSPECIFIED}
        const {cards} = cardsOf(async () => guest, async () => anaProgress)

        expect(await cards.playerInfo("account-guest")).toEqual({...guest, ...zeros})
    })

    it("shows zeros and no title for a player progression does not show", async () => {
        const {cards} = cardsOf(async () => ana, async () => undefined)

        expect(await cards.playerInfo("account-ana")).toEqual({...ana, ...zeros})
    })

    it("answers undefined for an account with no card", async () => {
        const {cards} = cardsOf(async () => undefined, async () => anaProgress)

        expect(await cards.playerInfo("account-gone")).toBeUndefined()
    })

    it("fails when either read fails", async () => {
        const failing = async (): Promise<never> => {
            throw new Error("down")
        }

        await expect(cardsOf(failing, async () => anaProgress).cards.playerInfo("account-ana")).rejects.toThrow("down")
        await expect(cardsOf(async () => ana, failing).cards.playerInfo("account-ana")).rejects.toThrow("down")
    })
})
