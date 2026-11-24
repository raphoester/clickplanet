import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {Submission} from "../../domain/flagSubmissions.ts"
import {useFlagForm} from "../flags/flagFormStore.ts"
import {useFlagSubmissions} from "../flags/submissionsStore.ts"
import {useScreen} from "../hud/screenStore.ts"
import {newestNews, useNews} from "../news/newsStore.ts"
import {PAID_READ_EVERY_MS, PAID_READ_FOR_MS, returnFromShop} from "./shopReturn.ts"

const approved: Submission = {id: "s-1", name: "Bavaria", flag: "", state: "approved", submittedAt: 1, decidedAt: 2, paid: false}
const opened: Submission = {...approved, state: "opened", flag: "r-k3p9x2ma0b"}

let refresh: ReturnType<typeof vi.fn>

beforeEach(() => {
    vi.useFakeTimers()
    refresh = vi.fn()
    useFlagSubmissions.setState({refresh, submissions: [approved], read: true})
})

afterEach(() => {
    vi.useRealTimers()
    useFlagSubmissions.setState(useFlagSubmissions.getInitialState(), true)
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useNews.setState(useNews.getInitialState(), true)
    useScreen.setState(useScreen.getInitialState(), true)
})

describe("returnFromShop", () => {
    it("says a payment, closes the flag's place and the shop, and reads the submissions every 3 seconds until the flag opens", () => {
        useFlagForm.getState().show()
        useScreen.getState().setShopOpen(true)
        const stop = returnFromShop("paid")

        expect(newestNews(useNews.getState())).toMatchObject({kind: "shop", back: "paid"})
        expect(useFlagForm.getState().shown).toBe(false)
        expect(useScreen.getState().shopOpen).toBe(false)

        vi.advanceTimersByTime(PAID_READ_EVERY_MS * 2)
        expect(refresh).toHaveBeenCalledTimes(2)

        useFlagSubmissions.setState({submissions: [opened]})
        vi.advanceTimersByTime(PAID_READ_EVERY_MS * 3)
        expect(refresh).toHaveBeenCalledTimes(2)
        stop()
    })

    it("stops reading after two minutes", () => {
        returnFromShop("paid")

        vi.advanceTimersByTime(PAID_READ_FOR_MS + PAID_READ_EVERY_MS * 5)

        expect(refresh).toHaveBeenCalledTimes(PAID_READ_FOR_MS / PAID_READ_EVERY_MS - 1)
    })

    it("waits for the first read of the submissions before it decides none waits", () => {
        useFlagSubmissions.setState({submissions: [], read: false})
        returnFromShop("paid")

        vi.advanceTimersByTime(PAID_READ_EVERY_MS * 2)

        expect(refresh).toHaveBeenCalledTimes(2)
    })

    it("says a page left, and reads nothing", () => {
        returnFromShop("cancelled")

        vi.advanceTimersByTime(PAID_READ_EVERY_MS * 3)

        expect(newestNews(useNews.getState())).toMatchObject({kind: "shop", back: "cancelled"})
        expect(refresh).not.toHaveBeenCalled()
    })
})
