import {useCallback, useEffect, useState} from "react"
import {sameTerritory, Territory, TerritoryPeek} from "../../domain/territory.ts"
import {Point} from "../../domain/tilePeek.ts"
import {LEAVE_GRACE_MS} from "../players/useTileHolder.ts"
import {SAMPLE_MS} from "../viewer/useLeaderboardFeed.ts"

export type TerritoryShown = {
    territory: Territory
    point: Point
    touch: boolean
    id: number
}

export function useTerritoryCard(
    pointing: {peek: TerritoryPeek, id: number} | undefined,
    territoryOf: (landmass: number) => Territory | undefined,
    showTerritory: (landmass: number | undefined) => void,
): {shown: TerritoryShown | undefined, hold: (held: boolean) => void, dismiss: () => void} {
    const [shown, setShown] = useState<TerritoryShown>()
    const [heldId, setHeldId] = useState<number>()

    useEffect(() => {
        if (!pointing) return
        const {peek: {landmass, point, touch}, id} = pointing
        const territory = territoryOf(landmass)
        if (territory) setShown({territory, point, touch, id})
    }, [pointing, territoryOf])

    const landmass = shown?.territory.landmass
    useEffect(() => {
        if (landmass === undefined) return
        showTerritory(landmass)
        const timer = setInterval(() => setShown((was) => {
            const territory = territoryOf(landmass)
            if (!was || !territory || sameTerritory(was.territory, territory)) return was
            return {...was, territory}
        }), SAMPLE_MS)
        return () => {
            clearInterval(timer)
            showTerritory(undefined)
        }
    }, [landmass, territoryOf, showTerritory])

    const id = shown?.id
    const held = id !== undefined && heldId === id
    useEffect(() => {
        if (pointing || held || id === undefined) return
        const timer = setTimeout(() => setShown(undefined), LEAVE_GRACE_MS)
        return () => clearTimeout(timer)
    }, [pointing, held, id])

    const hold = useCallback((on: boolean) => setHeldId(on ? id : undefined), [id])
    const dismiss = useCallback(() => setShown(undefined), [])

    return {shown, hold, dismiss}
}
