import {useEffect} from "react"
import {LandmassFortification, LandmassReader} from "../../backends/backend.ts"
import {PlayerInfoBackend} from "../../backends/player.ts"
import {deletedLineOf, playerLineOf} from "../players/tileHolder.ts"
import TerritoryCard from "./TerritoryCard.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {Fortifier, playersOf, useTerritoryPlayers} from "./territoryPlayersStore.ts"

const TOP_PLAYERS = 3

export type TerritoryCardPlaceProps = {
    shown?: TerritoryShown
    flag: string
    reader?: LandmassReader
    players?: PlayerInfoBackend
    onHold: (held: boolean) => void
    onClose: () => void
}

// The planet says which accounts hold the territory shown and who fortified it; the players' cards say who they are.
export default function TerritoryCardPlace({shown, flag, reader, players, onHold, onClose}: TerritoryCardPlaceProps) {
    const landmass = shown?.territory.landmass
    const read = useTerritoryPlayers(playersOf(landmass))
    const show = useTerritoryPlayers((territory) => territory.show)
    const forget = useTerritoryPlayers((territory) => territory.forget)

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
            show(landmass, {top: top.filter((player) => player !== undefined), fortifier})
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who holds the territory", e)
        })
        return () => asking.abort()
    }, [reader, players, landmass, show])

    // Nothing is kept once the place goes.
    useEffect(() => forget, [forget])

    if (!shown) return null
    return <TerritoryCard key={`territory-${shown.id}`}
                          shown={shown}
                          flag={flag}
                          players={read}
                          onHold={onHold}
                          onClose={onClose}/>
}

async function fortifierOf({accountId, countryCode, at}: LandmassFortification, players: PlayerInfoBackend): Promise<Fortifier | undefined> {
    if (!accountId) return {line: deletedLineOf(countryCode), at}
    const info = await players.playerInfo(accountId)
    return info && {line: playerLineOf(accountId, countryCode, info), at}
}
