// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {Countries} from "../domain/countries.ts"
import {COUNTRY_STORAGE_KEY} from "./countryPreference.ts"
import {useCountry, useKeepCountry} from "./countryStore.ts"

afterEach(() => {
    cleanup()
    useCountry.setState(useCountry.getInitialState(), true)
    localStorage.clear()
})

describe("useKeepCountry", () => {
    it("keeps nothing before the page renders, then the country played and each one chosen", () => {
        expect(localStorage.getItem(COUNTRY_STORAGE_KEY)).toBeNull()

        renderHook(() => useKeepCountry())
        expect(JSON.parse(localStorage.getItem(COUNTRY_STORAGE_KEY)!)).toEqual(useCountry.getState().country)

        act(() => useCountry.getState().choose(Countries.get("ie")!))
        expect(JSON.parse(localStorage.getItem(COUNTRY_STORAGE_KEY)!)).toMatchObject({code: "ie"})
    })
})
