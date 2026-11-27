// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {FlagSubmitter} from "../../backends/flags.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import {newestNews, useNews} from "../news/newsStore.ts"
import {useFlags} from "./flagsStore.ts"
import {useFlagSubmissions, useFlagSubmissionsFeed, useVerdictReads, VERDICT_READ_EVERY_MS, VERDICT_READ_FOR_MS, Who} from "./submissionsStore.ts"

const TOLD_KEY = "clickplanet-flag-submissions-told"

const settled = () => act(() => new Promise((resolve) => setTimeout(resolve, 0)))

const submission = (id: string, changes: Partial<Submission> = {}): Submission => ({
    id, name: "Bavaria", flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false, ...changes,
})

// Each flag a submission opened is held by its founder.
function submitterOf(...answers: Submission[][]): FlagSubmitter & {myFlags: ReturnType<typeof vi.fn>} {
    const myFlags = vi.fn(async () => {
        const submissions = answers.length > 1 ? answers.shift()! : answers[0]
        const held = submissions.filter(({state}) => state === "opened").map(({flag}) => ({code: flag}))
        return {submissions, held}
    })
    return {myFlags, uploadPicture: vi.fn(), submitFlag: vi.fn(), appealSubmission: vi.fn()}
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

        expect(submitter.myFlags).toHaveBeenCalledTimes(2)
        expect(useFlagSubmissions.getState().submissions.map(({state}) => state)).toEqual(["rejected"])
    })

    it("reads nothing for a guest, and forgets the list once the account signs out", async () => {
        const submitter = submitterOf([submission("a")])
        const view = feed(submitter, "guest")
        await settled()
        expect(submitter.myFlags).not.toHaveBeenCalled()
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

        act(() => useFlagSubmissions.setState({
            submissions: [submission("a", {state: "opened", flag: "r-k3p9x2ma0b"})], held: [{code: "r-k3p9x2ma0b"}],
        }))
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

    it("reads the submissions again when a flag the account holds closes or comes back, and for no other flag", async () => {
        flagsRead(false)
        const submitter = submitterOf([submission("a", {state: "opened", flag: BAVARIA})])
        feed(submitter)
        await settled()
        expect(submitter.myFlags).toHaveBeenCalledTimes(1)

        act(() => useFlags.setState(({catalog}) => ({catalog: catalog.withFlags([{code: "r-0000000002", name: "Tyrol", closed: true}])})))
        await settled()
        expect(submitter.myFlags).toHaveBeenCalledTimes(1)

        act(() => flagsRead(true))
        await settled()
        expect(submitter.myFlags).toHaveBeenCalledTimes(2)

        act(() => flagsRead(false))
        await settled()
        expect(submitter.myFlags).toHaveBeenCalledTimes(3)
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

    it("tells nothing of a flag the account no longer holds: another player claimed it", async () => {
        flagsRead(true)
        feed(submitterOf([submission("a", {state: "opened", flag: BAVARIA})]))
        await settled()
        useNews.setState(useNews.getInitialState(), true)

        act(() => useFlagSubmissions.setState({held: []}))
        act(() => flagsRead(false))

        expect(newestNews(useNews.getState())).toBeUndefined()
    })

    it("puts a submission sent first, before the next read says it", async () => {
        feed(submitterOf([submission("a", {state: "rejected"})]))
        await settled()

        act(() => useFlagSubmissions.getState().submitted(submission("b")))

        expect(useFlagSubmissions.getState().submissions.map(({id}) => id)).toEqual(["b", "a"])
    })

    it("puts a change sent in the place of the submission it replaced", async () => {
        feed(submitterOf([submission("a"), submission("x", {state: "rejected"})]))
        await settled()

        act(() => useFlagSubmissions.getState().submitted(submission("b"), "a"))

        expect(useFlagSubmissions.getState().submissions.map(({id}) => id)).toEqual(["b", "x"])
    })
})

describe("the verdict reads", () => {
    afterEach(() => vi.useRealTimers())

    it("read the submissions every 3 seconds while one waits, for two minutes at most", () => {
        vi.useFakeTimers()
        const refresh = vi.fn()
        useFlagSubmissions.setState({refresh})
        const {rerender, unmount} = renderHook(({waiting}: {waiting: boolean}) => useVerdictReads(waiting), {initialProps: {waiting: true}})

        vi.advanceTimersByTime(VERDICT_READ_EVERY_MS * 2)
        expect(refresh).toHaveBeenCalledTimes(2)

        rerender({waiting: false})
        vi.advanceTimersByTime(VERDICT_READ_EVERY_MS * 3)
        expect(refresh, "no read once it is decided").toHaveBeenCalledTimes(2)

        rerender({waiting: true})
        vi.advanceTimersByTime(VERDICT_READ_FOR_MS + VERDICT_READ_EVERY_MS * 5)
        expect(refresh).toHaveBeenCalledTimes(2 + VERDICT_READ_FOR_MS / VERDICT_READ_EVERY_MS - 1)
        unmount()
        useFlagSubmissions.setState(useFlagSubmissions.getInitialState(), true)
    })
})
