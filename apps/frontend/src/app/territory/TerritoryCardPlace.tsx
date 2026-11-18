import {useEffect, useMemo} from "react"
import {LandmassReader, ProvinceFortification, ProvinceReader} from "../../backends/backend.ts"
import {DayMissions} from "../../backends/missions.ts"
import {PlayerInfoBackend} from "../../backends/player.ts"
import {deletedLineOf, playerLineOf} from "../players/tileHolder.ts"
import TerritoryCard from "./TerritoryCard.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {Fortifier, fortifiedOf, topOf, useTerritoryPlayers} from "./territoryPlayersStore.ts"
import {useCountry} from "../countryStore.ts"

const TOP_PLAYERS = 3

export type TerritoryCardPlaceProps = {
    shown?: TerritoryShown
    reader: LandmassReader
    provinces: ProvinceReader
    players: PlayerInfoBackend
    missions?: DayMissions
    onHold: (held: boolean) => void
    onClose: () => void
}

// The planet says which accounts hold the territory shown, and who fortified the province its foot is about; the
// players' cards say who they are.
export default function TerritoryCardPlace({shown, reader, provinces, players, missions, onHold, onClose}: TerritoryCardPlaceProps) {
    const flag = useCountry((store) => store.country.code)
    const landmass = shown?.territory.landmass
    const province = shown?.territory.province
    const top = useTerritoryPlayers(topOf(landmass))
    const fortified = useTerritoryPlayers(fortifiedOf(province))
    const showTop = useTerritoryPlayers((territory) => territory.showTop)
    const showFortified = useTerritoryPlayers((territory) => territory.showFortified)
    const forget = useTerritoryPlayers((territory) => territory.forget)
    const read = useMemo(() => top === undefined && fortified === undefined
        ? undefined
        : {top: top ?? [], fortifier: fortified?.by}, [top, fortified])

    useEffect(() => {
        if (landmass === undefined) return

        const asking = new AbortController()
        const ask = async () => {
            const {holders} = await reader.landmass(landmass, asking.signal)
            const top = await Promise.all(holders.slice(0, TOP_PLAYERS).map(async ({accountId, countryCode, tiles}) => {
                const info = await players.playerInfo(accountId)
                return info && {line: playerLineOf(accountId, countryCode, info), tiles}
            }))
            if (asking.signal.aborted) return
            showTop(landmass, top.filter((player) => player !== undefined))
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who holds the territory", e)
        })
        return () => asking.abort()
    }, [reader, players, landmass, showTop])

    useEffect(() => {
        if (province === undefined) return

        const asking = new AbortController()
        const ask = async () => {
            const {lastFortification} = await provinces.province(province, asking.signal)
            const by = lastFortification && await fortifierOf(lastFortification, players)
            if (asking.signal.aborted) return
            showFortified(province, {by})
        }
        ask().catch((e) => {
            if (!asking.signal.aborted) console.error("could not read who fortified the province", e)
        })
        return () => asking.abort()
    }, [provinces, players, province, showFortified])

    // Nothing is kept once the place goes.
    useEffect(() => forget, [forget])

    if (!shown) return null
    return <TerritoryCard key={`territory-${shown.id}`}
                          shown={shown}
                          flag={flag}
                          players={read}
                          missions={missions}
                          onHold={onHold}
                          onClose={onClose}/>
}

async function fortifierOf({accountId, countryCode, at}: ProvinceFortification, players: PlayerInfoBackend): Promise<Fortifier | undefined> {
    if (!accountId) return {line: deletedLineOf(countryCode), at}
    const info = await players.playerInfo(accountId)
    return info && {line: playerLineOf(accountId, countryCode, info), at}
}
