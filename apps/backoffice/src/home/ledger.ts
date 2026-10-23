import {Timestamp} from "@bufbuild/protobuf"
import {GetLedgerResponse, LedgerChange} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Territory} from "../gen/grpc/planet/v1/internal_pb.ts"
import {Slot} from "./activity.ts"

export type Window = {since: Date, until: Date, count: number, step: number}

// Whole slots: the last one starts on the step before until, and is still filling.
export function windowOf(until: Date, hours: number, step: number): Window {
    const count = Math.ceil((hours * 3600_000) / step)
    const last = Math.floor(until.getTime() / step) * step
    return {since: new Date(last - (count - 1) * step), until, count, step}
}

// The tile changed hands: a click, a spread, an enclosure or a bomb. A strike or a shield did not.
function moved(change: LedgerChange): boolean {
    return change.update !== undefined && change.update.countryId !== change.update.previousCountryId
}

function millisOf(at: Timestamp | undefined): number {
    return at === undefined ? 0 : Number(at.seconds) * 1000 + at.nanos / 1e6
}

export function slotsOf(ledger: GetLedgerResponse, {since, until, count, step}: Window): Slot[] {
    const slots: Slot[] = Array.from({length: count}, (_, i) => {
        const start = new Date(since.getTime() + i * step)
        const end = new Date(start.getTime() + step)
        return {start, end, clicks: 0, tiles: 0, players: 0, filling: end > until}
    })
    const takers = slots.map(() => new Set<string>())

    for (const change of ledger.changes) {
        const slot = Math.floor((millisOf(change.at) - since.getTime()) / step)
        if (slot < 0 || slot >= count) continue
        if (change.update?.clicked === true) slots[slot].clicks++
        if (moved(change)) {
            slots[slot].tiles++
            takers[slot].add(change.by)
        }
    }

    takers.forEach((who, i) => slots[i].players = who.size)
    return slots
}

export function playersOf(ledger: GetLedgerResponse): number {
    return new Set(ledger.changes.filter(moved).map(change => change.by)).size
}

// How often each tile changed hands, by tile id.
export function changesOf(ledger: GetLedgerResponse, tiles: number): Uint32Array {
    const changes = new Uint32Array(tiles + 1)
    for (const change of ledger.changes) {
        const tile = change.update?.tileId ?? 0
        if (moved(change) && tile <= tiles) changes[tile]++
    }
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
    for (const change of ledger.changes) {
        if (change.update === undefined || !moved(change)) continue
        if (change.update.countryId !== "") rowOf(change.update.countryId).won++
        if (change.update.previousCountryId !== "") rowOf(change.update.previousCountryId).lost++
    }

    return [...rows.values()].sort((a, b) => b.held - a.held || b.won - a.won || a.countryId.localeCompare(b.countryId))
}
