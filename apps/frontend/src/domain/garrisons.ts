import type {OwnerChange} from "./tileOwnership.ts"

export type Garrison = {
    tile: number
    country: string
    defenders: number
}

export type GarrisonChange = {
    tile: number
    defenders: number
    was: number
}

export type ClickOutcome = "taken" | "defended" | "unchanged"

export function outcomeOf(owner: string | undefined, flag: string, defenders: number): ClickOutcome {
    if (owner === flag) return "unchanged"
    return defenders > 0 ? "defended" : "taken"
}

export type Placement = "place" | "full" | "click"

export function placementOf(owner: string | undefined, flag: string, defenders: number, most: number | undefined): Placement {
    if (owner !== flag) return "click"
    return most !== undefined && most > 0 && defenders >= most ? "full" : "place"
}

export class TileGarrisons {
    private readonly held = new Map<number, {country: string, defenders: number}>()
    private early: Garrison[] | undefined = []

    constructor(private readonly ownerOf: (tile: number) => string | undefined) {
    }

    public defendersOf(tile: number): number {
        return this.held.get(tile)?.defenders ?? 0
    }

    public applyLoaded(garrisons: readonly Garrison[]): GarrisonChange[] {
        const early = this.early ?? []
        this.early = undefined
        return [...garrisons, ...early].flatMap((garrison) => this.set(garrison))
    }

    public apply(garrison: Garrison): GarrisonChange[] {
        if (this.early) {
            this.early.push(garrison)
            return []
        }
        return this.set(garrison)
    }

    public followOwners(changes: readonly OwnerChange[]): GarrisonChange[] {
        const dropped: GarrisonChange[] = []
        for (const {tile, country} of changes) {
            const garrison = this.held.get(tile)
            if (!garrison || garrison.country === country) continue
            this.held.delete(tile)
            dropped.push({tile, defenders: 0, was: garrison.defenders})
        }
        return dropped
    }

    private set({tile, country, defenders}: Garrison): GarrisonChange[] {
        if (this.ownerOf(tile) !== country) return []

        const was = this.defendersOf(tile)
        if (defenders === was) return []

        if (defenders > 0) this.held.set(tile, {country, defenders})
        else this.held.delete(tile)
        return [{tile, defenders, was}]
    }
}
