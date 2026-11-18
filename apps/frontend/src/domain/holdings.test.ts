import {describe, expect, it} from "vitest"
import type {Counts} from "../backends/backend.ts"
import {Holdings} from "./holdings.ts"

type Rows = Record<number, Record<string, number>>

const rowsOf = (rows: Rows) => new Map(Object.entries(rows).map(([ground, flags]) => [Number(ground), new Map(Object.entries(flags))]))

const countsAt = (seq: number, countries: Record<string, number>, landmasses: Rows = {}, provinces: Rows = {}): Counts => ({
    version: {epoch: 1_790_000_000, seq},
    countries: new Map(Object.entries(countries)),
    landmasses: rowsOf(landmasses),
    provinces: rowsOf(provinces),
})

const board = (holdings: Holdings) => Object.fromEntries(holdings.counts())

const row = (holdings: Holdings, landmass: number) => Object.fromEntries(holdings.holdingsOf(landmass))

describe("Holdings", () => {
    it("takes a read whole, and says which rows moved", () => {
        const holdings = new Holdings()

        expect(holdings.read(countsAt(5, {fr: 3, de: 1}, {7: {fr: 3, de: 1}}))).toEqual({countries: true, landmasses: [7], provinces: []})
        expect(board(holdings)).toEqual({fr: 3, de: 1})
        expect(row(holdings, 7)).toEqual({fr: 3, de: 1})
        expect(holdings.tilesOf("fr")).toBe(3)
        expect(row(holdings, 8)).toEqual({})
    })

    it("takes the rows a change lists, a flag at 0 off the board, and leaves the others", () => {
        const holdings = new Holdings()
        holdings.read(countsAt(5, {fr: 3, de: 1}, {7: {fr: 3, de: 1}, 8: {it: 2}}))

        const moved = holdings.change(countsAt(6, {fr: 4, de: 0}, {7: {fr: 4}}))

        expect(moved).toEqual({countries: true, landmasses: [7], provinces: []})
        expect(board(holdings)).toEqual({fr: 4})
        expect(row(holdings, 7)).toEqual({fr: 4})
        expect(row(holdings, 8)).toEqual({it: 2})
    })

    it("lets no older copy bring back a row, at 0 or not", () => {
        const holdings = new Holdings()
        holdings.change(countsAt(9, {fr: 0, de: 2}, {7: {}}))

        expect(holdings.read(countsAt(8, {fr: 5, de: 9}, {7: {fr: 5}}))).toEqual({countries: false, landmasses: [], provinces: []})
        expect(board(holdings)).toEqual({de: 2})
        expect(row(holdings, 7)).toEqual({})
    })

    it("drops what a newer read no longer lists, and keeps what the stream said after it", () => {
        const holdings = new Holdings()
        holdings.read(countsAt(5, {fr: 3, de: 1}, {7: {fr: 3}, 8: {de: 1}}))
        holdings.change(countsAt(12, {it: 1}, {9: {it: 1}}))

        holdings.read(countsAt(10, {fr: 3}, {7: {fr: 3}}))

        expect(board(holdings)).toEqual({fr: 3, it: 1})
        expect(row(holdings, 8)).toEqual({})
        expect(row(holdings, 9)).toEqual({it: 1})
    })

    it("says nothing moved when a read repeats what it holds", () => {
        const holdings = new Holdings()
        holdings.read(countsAt(5, {fr: 3}, {7: {fr: 3}}))

        expect(holdings.read(countsAt(6, {fr: 3}, {7: {fr: 3}}))).toEqual({countries: false, landmasses: [], provinces: []})
    })

    it("takes anything after a restart: the versions held compare with none of the new start", () => {
        const holdings = new Holdings()
        holdings.read(countsAt(500, {fr: 3}, {7: {fr: 3}}))

        holdings.forget()
        holdings.read(countsAt(2, {de: 1}, {7: {de: 1}}))

        expect(board(holdings)).toEqual({de: 1})
        expect(row(holdings, 7)).toEqual({de: 1})
    })

    it("names the leader of a landmass, ties on the flag's code", () => {
        const holdings = new Holdings()
        holdings.read(countsAt(5, {}, {7: {fr: 2, de: 2, it: 1}}))

        expect(holdings.leaderOf(7)).toEqual({holder: "de", held: 2})
        expect(holdings.leaderOf(8)).toBeUndefined()
    })

    it("keeps each province's row as each landmass's, apart from them", () => {
        const holdings = new Holdings()

        expect(holdings.read(countsAt(5, {fr: 3}, {7: {fr: 3}}, {7: {de: 2}, 12: {fr: 3}}))).toEqual({countries: true, landmasses: [7], provinces: [7, 12]})
        expect(Object.fromEntries(holdings.holdingsOfProvince(7))).toEqual({de: 2})
        expect(holdings.leaderOfProvince(12)).toEqual({holder: "fr", held: 3})
        expect(row(holdings, 7)).toEqual({fr: 3})

        expect(holdings.change(countsAt(6, {fr: 4}, {}, {12: {fr: 4}, 7: {}}))).toEqual({countries: true, landmasses: [], provinces: [7, 12]})
        expect(holdings.leaderOfProvince(7)).toBeUndefined()
        expect(holdings.read(countsAt(4, {}, {}, {12: {it: 9}}))).toEqual({countries: false, landmasses: [], provinces: []})
        expect(holdings.leaderOfProvince(12)).toEqual({holder: "fr", held: 4})
    })

    it("drops a province a newer read no longer lists, and takes any after a restart", () => {
        const holdings = new Holdings()
        holdings.read(countsAt(5, {}, {}, {3: {fr: 1}, 4: {de: 1}}))

        expect(holdings.read(countsAt(6, {}, {}, {4: {de: 1}})).provinces).toEqual([3])
        holdings.forget()
        holdings.read(countsAt(1, {}, {}, {4: {it: 2}}))
        expect(Object.fromEntries(holdings.holdingsOfProvince(4))).toEqual({it: 2})
    })
})
