import {afterEach, describe, expect, it} from "vitest"
import {TerritoryPlayers, playersOf, useTerritoryPlayers} from "./territoryPlayersStore.ts"

const NOBODY: TerritoryPlayers = {top: [], fortifier: undefined}

afterEach(() => useTerritoryPlayers.setState({landmass: undefined, players: undefined}))

describe("a territory's players", () => {
    it("are only the territory's they were read for", () => {
        useTerritoryPlayers.getState().show(12, NOBODY)

        expect(playersOf(12)(useTerritoryPlayers.getState())).toBe(NOBODY)
        expect(playersOf(13)(useTerritoryPlayers.getState())).toBeUndefined()
        expect(playersOf(undefined)(useTerritoryPlayers.getState())).toBeUndefined()
    })

    it("are nobody's once forgotten", () => {
        useTerritoryPlayers.getState().show(12, NOBODY)

        useTerritoryPlayers.getState().forget()

        expect(playersOf(12)(useTerritoryPlayers.getState())).toBeUndefined()
    })
})
