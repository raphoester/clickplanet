import {Timestamp} from "@bufbuild/protobuf"
import {GetLedgerResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Territory} from "../gen/grpc/planet/v1/internal_pb.ts"
import {Slot} from "./activity.ts"

// A window of whole slots: the last one starts on the step before until, and is still filling.
export function windowOf(until: Date, hours: number, step: number): {since: Date, count: number} {
    const count = Math.ceil((hours * 3600_000) / step)
    const last = Math.floor(until.getTime() / step) * step
    return {since: new Date(last - (count - 1) * step), count}
}

// The tile changed hands: a click, a spread, an enclosure or a bomb. A strike or a shield did not.
function moved(ledger: GetLedgerResponse, i: number): boolean {
    return ledger.owner[i] !== ledger.previous[i]
}

function whoOf(ledger: GetLedgerResponse, i: number): string {
    return ledger.account[i] !== 0 ? `account ${ledger.account[i]}` : `scope ${ledger.scope[i]}`
}

export function slotsOf(ledger: GetLedgerResponse, count: number, step: number): Slot[] {
    const since = dateOf(ledger.since)
    const until = dateOf(ledger.until)
    const slots: Slot[] = Array.from({length: count}, (_, i) => {
        const start = new Date(since.getTime() + i * step)
        const end = new Date(start.getTime() + step)
        return {start, end, clicks: 0, tiles: 0, players: 0, filling: end > until}
    })
    const takers = slots.map(() => new Set<string>())

    ledger.at.forEach((at, i) => {
        const slot = Math.floor(at / step)
        if (slot < 0 || slot >= count) return
        if (ledger.clicked[i] === 1) slots[slot].clicks++
        if (moved(ledger, i)) {
            slots[slot].tiles++
            takers[slot].add(whoOf(ledger, i))
        }
    })

    takers.forEach((who, i) => slots[i].players = who.size)
    return slots
}

export function playersOf(ledger: GetLedgerResponse): number {
    const who = new Set<string>()
    ledger.at.forEach((_, i) => {
        if (moved(ledger, i)) who.add(whoOf(ledger, i))
    })
    return who.size
}

// How often each tile changed hands, by tile id.
export function changesOf(ledger: GetLedgerResponse, tiles: number): Uint32Array {
    const changes = new Uint32Array(tiles + 1)
    ledger.tile.forEach((tile, i) => {
        if (moved(ledger, i) && tile <= tiles) changes[tile]++
    })
    return changes
}

export type CountryRow = {countryId: string, held: number, won: number, lost: number}

export type Order = "held" | "won" | "lost"

export function countriesBy(countries: CountryRow[], order: Order): CountryRow[] {
    if (order === "held") return countries
    return [...countries].sort((a, b) => b[order] - a[order] || b.held - a.held)
}

export function heldOf(countries: CountryRow[]): number {
    return countries.reduce((sum, country) => sum + country.held, 0)
}

// What each country holds now, and the tiles taken for it and from it in the window.
export function countriesOf(ledger: GetLedgerResponse, territories: Territory[]): CountryRow[] {
    const rows = new Map<string, CountryRow>()
    const rowOf = (countryId: string) => {
        let row = rows.get(countryId)
        if (row === undefined) {
            row = {countryId, held: 0, won: 0, lost: 0}
            rows.set(countryId, row)
        }
        return row
    }

    for (const territory of territories) rowOf(territory.countryId).held = territory.tiles
    ledger.at.forEach((_, i) => {
        if (!moved(ledger, i)) return
        const owner = ledger.countries[ledger.owner[i]] ?? ""
        const previous = ledger.countries[ledger.previous[i]] ?? ""
        if (owner !== "") rowOf(owner).won++
        if (previous !== "") rowOf(previous).lost++
    })

    return [...rows.values()].sort((a, b) => b.held - a.held || b.won - a.won || a.countryId.localeCompare(b.countryId))
}

function dateOf(at: Timestamp | undefined): Date {
    return at === undefined ? new Date(0) : at.toDate()
}
