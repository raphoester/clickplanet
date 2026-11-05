export const SLOT_COUNT = 8

export const EVICTION_MARGIN = 1.2

export type Slots = readonly (string | undefined)[]

// Which flags are drawn large: those holding the biggest landmasses, one slot each, kept while they hold their place.
export class SlotRanking {
    private readonly slots: (string | undefined)[]

    constructor(count = SLOT_COUNT, private readonly margin = EVICTION_MARGIN) {
        this.slots = new Array<string | undefined>(count).fill(undefined)
    }

    rank(needs: ReadonlyMap<string, number>): Slots {
        const needOf = (code: string) => needs.get(code) ?? 0

        for (let slot = 0; slot < this.slots.length; slot++) {
            const code = this.slots[slot]
            if (code !== undefined && !needs.has(code)) this.slots[slot] = undefined
        }

        const waiting = Array.from(needs.keys())
            .filter((code) => !this.slots.includes(code))
            .sort((a, b) => needOf(b) - needOf(a) || a.localeCompare(b))

        for (const code of waiting) {
            const free = this.slots.indexOf(undefined)
            if (free >= 0) {
                this.slots[free] = code
                continue
            }
            const weakest = this.weakest(needOf)
            if (needOf(code) <= needOf(this.slots[weakest]!) * this.margin) break
            this.slots[weakest] = code
        }
        return [...this.slots]
    }

    private weakest(needOf: (code: string) => number): number {
        let weakest = 0
        for (let slot = 1; slot < this.slots.length; slot++) {
            if (needOf(this.slots[slot]!) < needOf(this.slots[weakest]!)) weakest = slot
        }
        return weakest
    }
}
