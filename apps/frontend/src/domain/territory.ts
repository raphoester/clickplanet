import type {Target} from "./fortifyTarget.ts"
import type {Point} from "./tilePeek.ts"

export type TerritoryPeek = {
    landmass: number
    point: Point
    touch: boolean
}

export type Holding = {
    flag: string
    tiles: number
}

export type Territory = {
    landmass: number
    name: string
    ground: string | undefined
    size: number
    holdings: Holding[]
    fortifiedBy: string | undefined
    target: Target | undefined
}

// Ties break on the flag's code, as on the board, so equal rows stop swapping.
export function rankHoldings(held: ReadonlyMap<string, number>): Holding[] {
    const holdings: Holding[] = []
    held.forEach((tiles, flag) => {
        if (tiles > 0) holdings.push({flag, tiles})
    })
    return holdings.sort((a, b) => b.tiles - a.tiles || a.flag.localeCompare(b.flag))
}

export function freeTilesOf({size, holdings}: Territory): number {
    return Math.max(0, holdings.reduce((free, {tiles}) => free - tiles, size))
}

// Whole percents: a flag holding a few tiles is never 0%, and one short of all of it is never 100%.
export function sharePercent(tiles: number, size: number): string {
    if (size <= 0 || tiles <= 0) return "0"
    if (tiles >= size) return "100"
    const percent = Math.round(tiles / size * 100)
    if (percent < 1) return "<1"
    return String(Math.min(percent, 99))
}

export function sameTerritory(a: Territory, b: Territory): boolean {
    return a.landmass === b.landmass
        && a.size === b.size
        && a.fortifiedBy === b.fortifiedBy
        && a.target?.flag === b.target?.flag
        && a.target?.missing === b.target?.missing
        && a.holdings.length === b.holdings.length
        && a.holdings.every((holding, i) => holding.flag === b.holdings[i].flag && holding.tiles === b.holdings[i].tiles)
}
