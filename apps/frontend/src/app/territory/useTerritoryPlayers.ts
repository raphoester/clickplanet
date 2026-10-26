import {useEffect, useState} from "react"
import {LandmassFortification, LandmassReader} from "../../backends/backend.ts"
import {PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import {deletedLineOf, playerLineOf} from "../players/useTileHolder.ts"

export const TOP_PLAYERS = 3

export type TerritoryPlayer = {
    line: PlayerLine
    tiles: number
}

export type Fortifier = {
    line: PlayerLine
    at: number
}

export type TerritoryPlayers = {
    top: TerritoryPlayer[]
    fortifier: Fortifier | undefined
}

// The planet says which accounts hold the territory and who fortified it; the players' cards say who they are.
export function useTerritoryPlayers(
    reader: LandmassReader | undefined,
    players: PlayerInfoBackend | undefined,
    landmass: number | undefined,
): TerritoryPlayers | undefined {
    const [read, setRead] = useState<{landmass: number, players: TerritoryPlayers}>()

    useEffect(() => {
        if (landmass === undefined || !reader || !players) return

        const asking = new AbortController()
        const ask = async () => {
            const {holders, lastFortification} = await reader.landmass(landmass, asking.signal)
            const [top, fortifier] = await Promise.all([
                Promise.all(holders.slice(0, TOP_PLAYERS).map(async ({accountId, countryCode, tiles}) => {
                    const info = await players.playerInfo(accountId)
                    return info && {line: playerLineOf(accountId, countryCode, info), tiles}
                })),
                lastFortification && fortifierOf(lastFortification, players),
            ])
            if (asking.signal.aborted) return
            setRead({landmass, players: {top: top.filter((player) => player !== undefined), fortifier}})
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who holds the territory", e)
        })
        return () => asking.abort()
    }, [reader, players, landmass])

    return read && read.landmass === landmass ? read.players : undefined
}

async function fortifierOf({accountId, countryCode, at}: LandmassFortification, players: PlayerInfoBackend): Promise<Fortifier | undefined> {
    if (!accountId) return {line: deletedLineOf(countryCode), at}
    const info = await players.playerInfo(accountId)
    return info && {line: playerLineOf(accountId, countryCode, info), at}
}
