// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {FlagSubmitter} from "../../backends/flags.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import {newestNews, useNews} from "../news/newsStore.ts"
import {useFlags} from "./flagsStore.ts"
import {useFlagSubmissions, useFlagSubmissionsFeed, Who} from "./submissionsStore.ts"

const TOLD_KEY = "clickplanet-flag-submissions-told"

const settled = () => act(() => new Promise((resolve) => setTimeout(resolve, 0)))

const submission = (id: string, changes: Partial<Submission> = {}): Submission => ({
    id, name: "Bavaria", flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, ...changes,
})

function submitterOf(...answers: Submission[][]): FlagSubmitter & {mySubmissions: ReturnType<typeof vi.fn>} {
    const mySubmissions = vi.fn(async () => answers.length > 1 ? answers.shift()! : answers[0])
    return {mySubmissions, uploadPicture: vi.fn(), submitFlag: vi.fn()}
}

function feed(submitter: FlagSubmitter, who: Who = "player") {
    return renderHook(({who}: {who: Who}) => useFlagSubmissionsFeed(submitter, who), {initialProps: {who}})
}

const BAVARIA = "r-k3p9x2ma0b"

const flagsRead = (closed: boolean) => useFlags.setState({
    read: true,
    catalog: FlagCatalog.BUILT_IN.withFlags([{code: BAVARIA, name: "Bavaria", closed}]),
})

beforeEach(() => {
    useFlags.setState({read: true})
})

afterEach(() => {
    cleanup()
    window.localStorage.clear()
    useNews.setState(useNews.getInitialState(), true)
    useFlags.setState({openings: 0, read: false, catalog: FlagCatalog.BUILT_IN})
})

describe("the flag submissions feed", () => {
    it("reads a signed-in account's submissions at once, and again each time the flags stream opens", async () => {
        const submitter = submitterOf([submission("a")], [submission("a", {state: "rejected"})])
        feed(submitter)
        await settled()

        expect(useFlagSubmissions.getState().submissions.map(({state}) => state)).toEqual(["waiting"])
        expect(useFlagSubmissions.getState().who).toBe("player")

        act(() => useFlags.setState(({openings}) => ({openings: openings + 1})))
        await settled()

        expect(submitter.mySubmissions).toHaveBeenCalledTimes(2)
        expect(useFlagSubmissions.getState().submissions.map(({state}) => state)).toEqual(["rejected"])
    })

    it("reads nothing for a guest, and forgets the list once the account signs out", async () => {
        const submitter = submitterOf([submission("a")])
        const view = feed(submitter, "guest")
        await settled()
        expect(submitter.mySubmissions).not.toHaveBeenCalled()
        expect(useFlagSubmissions.getState().who).toBe("guest")

        view.rerender({who: "player"})
        await settled()
        expect(useFlagSubmissions.getState().submissions).toHaveLength(1)

        view.rerender({who: "guest"})
        expect(useFlagSubmissions.getState().submissions).toEqual([])
    })

    it("tells a decision on the news line once, remembered by submission across loads", async () => {
        const approved = [submission("a", {state: "approved"})]
        const first = feed(submitterOf(approved))
        await settled()

        const told = newestNews(useNews.getState())
        expect(told?.kind === "submission" && told.submission.state).toBe("approved")
        expect(JSON.parse(window.localStorage.getItem(TOLD_KEY) ?? "{}")).toEqual({a: "approved"})

        first.unmount()
        useNews.setState(useNews.getInitialState(), true)
        feed(submitterOf(approved))
        await settled()
        expect(newestNews(useNews.getState())).toBeUndefined()

        act(() => useFlagSubmissions.setState({submissions: [submission("a", {state: "opened", flag: "r-k3p9x2ma0b"})]}))
        const opened = newestNews(useNews.getState())
        expect(opened?.kind === "submission" && opened.submission.state).toBe("opened")
    })

    it("tells a flag the account opened closed, once, and nothing before the list of flags is read", async () => {
        useFlags.setState({read: false})
        feed(submitterOf([submission("a", {state: "opened", flag: BAVARIA})]))
        await settled()
        expect(newestNews(useNews.getState())).toBeUndefined()

        act(() => flagsRead(false))
        const opened = newestNews(useNews.getState())
        expect(opened?.kind === "submission" && opened.submission.state).toBe("opened")

        useNews.setState(useNews.getInitialState(), true)
        act(() => flagsRead(true))
        const closed = newestNews(useNews.getState())
        expect(closed?.kind === "submission" && closed.submission.state).toBe("closed")
        expect(JSON.parse(window.localStorage.getItem(TOLD_KEY) ?? "{}")).toEqual({a: "closed"})

        useNews.setState(useNews.getInitialState(), true)
        act(() => flagsRead(true))
        expect(newestNews(useNews.getState())).toBeUndefined()
    })

    it("reads the submissions again when a flag the account founded closes or comes back, and for no other flag", async () => {
        flagsRead(false)
        const submitter = submitterOf([submission("a", {state: "opened", flag: BAVARIA})])
        feed(submitter)
        await settled()
        expect(submitter.mySubmissions).toHaveBeenCalledTimes(1)

        act(() => useFlags.setState(({catalog}) => ({catalog: catalog.withFlags([{code: "r-0000000002", name: "Tyrol", closed: true}])})))
        await settled()
        expect(submitter.mySubmissions).toHaveBeenCalledTimes(1)

        act(() => flagsRead(true))
        await settled()
        expect(submitter.mySubmissions).toHaveBeenCalledTimes(2)

        act(() => flagsRead(false))
        await settled()
        expect(submitter.mySubmissions).toHaveBeenCalledTimes(3)
    })

    it("tells nothing for a submission that waits, and nothing when the browser keeps nothing", async () => {
        feed(submitterOf([submission("w")]))
        await settled()
        expect(newestNews(useNews.getState())).toBeUndefined()

        cleanup()
        vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
            throw new Error("blocked")
        })
        feed(submitterOf([submission("a", {state: "rejected"})]))
        await settled()
        expect(newestNews(useNews.getState())).toBeUndefined()
        vi.restoreAllMocks()
    })

    it("puts a submission sent first, before the next read says it", async () => {
        feed(submitterOf([submission("a", {state: "rejected"})]))
        await settled()

        act(() => useFlagSubmissions.getState().submitted(submission("b")))

        expect(useFlagSubmissions.getState().submissions.map(({id}) => id)).toEqual(["b", "a"])
    })
})
