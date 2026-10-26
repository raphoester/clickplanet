import {describe, expect, it} from "vitest"
import {FortifyTargets, Holdings} from "./fortifyTargets.ts"

// Landmass 1 is tiles 1 to 4; landmass 2 is tile 5. The leader comes from the owners, as the server counts them.
function island(owners: Record<number, string>, known = new Set([1, 2, 3, 4, 5])) {
    const landmassOf = (tile: number) => (tile <= 4 ? 1 : tile === 5 ? 2 : 0)
    const holdings: Holdings = {
        sizeOf: (landmass) => (landmass === 1 ? 4 : landmass === 2 ? 1 : 0),
        leaderOf: (landmass) => {
            const held = new Map<string, number>()
            for (const [tile, flag] of Object.entries(owners)) {
                if (landmassOf(Number(tile)) === landmass) held.set(flag, (held.get(flag) ?? 0) + 1)
            }
            const [holder, count] = [...held].sort((a, b) => b[1] - a[1])[0] ?? []
            return holder === undefined ? undefined : {holder, held: count}
        },
        ownerOf: (tile) => (known.has(tile) ? owners[tile] : undefined),
        known: (tile) => known.has(tile),
        tilesOf: (landmass) => (landmass === 1 ? [1, 2, 3, 4] : landmass === 2 ? [5] : []),
        landmassOf,
    }
    const pulsing = new Set<number>()
    const targets = new FortifyTargets(holdings, (tiles, on) => tiles.forEach((tile) => on ? pulsing.add(tile) : pulsing.delete(tile)))
    return {owners, known, targets, pulsing}
}

describe("FortifyTargets", () => {
    it("pulses the tiles a leader close to the whole still misses, and says it entered", () => {
        const {targets, pulsing} = island({1: "us", 2: "us", 3: "us", 4: "ca"})

        const entered = targets.lockAll(new Map(), 3)

        expect(entered).toEqual([{landmass: 1, target: {flag: "us", missing: 1}}])
        expect([...pulsing]).toEqual([4])
    })

    it("follows each tile taken while the leader stays close", () => {
        const {owners, targets, pulsing} = island({1: "us", 2: "us", 3: "ca", 4: "ca"})
        targets.lockAll(new Map(), 3)
        expect(pulsing.size).toBe(0)

        owners[3] = "us"
        expect(targets.touch([3])).toEqual([{landmass: 1, target: {flag: "us", missing: 1}}])
        expect([...pulsing]).toEqual([4])

        owners[3] = "ca"
        owners[2] = "us"
        expect(targets.touch([3, 2])).toEqual([])
        expect(pulsing.size).toBe(0)
    })

    it("stops pulsing once the landmass is fortified by its leader", () => {
        const {owners, targets, pulsing} = island({1: "us", 2: "us", 3: "us", 4: "ca"})
        targets.lockAll(new Map(), 3)

        owners[4] = "us"
        targets.touch([4])
        targets.lock(1, "us")

        expect(pulsing.size).toBe(0)
        expect(targets.targetOf(1)).toBeUndefined()
    })

    it("never pulses for the flag that fortified it last", () => {
        const {targets, pulsing} = island({1: "us", 2: "us", 3: "us", 4: "ca"})

        targets.lockAll(new Map([[1, "us"]]), 3)

        expect(pulsing.size).toBe(0)
    })

    it("takes its target from the counts, and pulses only the tiles whose owner is read", () => {
        const {known, targets, pulsing} = island({1: "us", 2: "us", 3: "us", 4: "ca"}, new Set())

        expect(targets.lockAll(new Map(), 3)).toEqual([{landmass: 1, target: {flag: "us", missing: 1}}])
        expect(pulsing.size).toBe(0)

        for (const tile of [1, 2, 3, 4]) known.add(tile)
        targets.touch([1, 2, 3, 4])

        expect([...pulsing]).toEqual([4])
    })

    it("follows the counts as they change", () => {
        const {owners, targets, pulsing} = island({1: "us", 2: "us", 3: "ca", 4: "ca"})
        targets.lockAll(new Map(), 3)

        owners[3] = "us"
        expect(targets.recount([1])).toEqual([{landmass: 1, target: {flag: "us", missing: 1}}])
        expect([...pulsing]).toEqual([4])

        owners[4] = "us"
        targets.recount([1])
        expect(pulsing.size).toBe(0)
    })
})
