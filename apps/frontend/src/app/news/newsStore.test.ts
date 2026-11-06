import {afterEach, describe, expect, it} from "vitest"
import {BombDrop} from "../../backends/backend.ts"
import {Fortified} from "../../domain/fortify.ts"
import {newestNews, useNews} from "./newsStore.ts"

const DROP: BombDrop = {tile: 12, point: {x: 0, y: 0, z: 1}, countryId: "fr", radius: 0.03, cleared: [12, 13], struck: [], version: {epoch: 1, seq: 4}}
const CORSICA: Fortified = {fortification: {landmass: 257, countryId: "bg", tile: 107526, version: {epoch: 1, seq: 9}}, name: "Corsica", tiles: 16, news: true}

const newest = () => newestNews(useNews.getState())

afterEach(() => useNews.setState(useNews.getInitialState(), true))

describe("useNews", () => {
    it("shows the newest news, then the one before once that is done", () => {
        const news = useNews.getState()
        news.tellBomb(DROP, "Normandy")
        news.tellFortified(CORSICA)
        expect(newest()).toMatchObject({kind: "fortify", fortified: CORSICA})

        news.dismiss(newest()!)
        expect(newest()).toMatchObject({kind: "bomb", drop: DROP, land: "Normandy"})

        news.dismiss(newest()!)
        expect(newest()).toBeUndefined()
    })

    it("keeps the last of each kind only", () => {
        const news = useNews.getState()
        news.tellBomb(DROP, undefined)
        news.tellAttack()
        news.tellBomb({...DROP, countryId: "de"}, undefined)

        expect(useNews.getState().told.map((told) => told.kind)).toEqual(["attack", "bomb"])
        expect(newest()).toMatchObject({kind: "bomb", drop: {countryId: "de"}})
    })

    it("tells a fortification only when it is news", () => {
        useNews.getState().tellFortified({...CORSICA, news: false})

        expect(newest()).toBeUndefined()
    })
})
