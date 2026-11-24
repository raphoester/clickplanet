import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import type {FlagPicture} from "../domain/flags.ts"
import {FakeFlagsBackend} from "./fakeFlagsBackend.ts"
import {FakeFlagSubmissions} from "./fakeFlagSubmissions.ts"
import {FlagHandlers, SubmissionRefusedError} from "./flags.ts"

const PICTURE: FlagPicture = {url: "blob:upload", width: 600, height: 400, stretch: false, focusX: 0.5, focusY: 0.5}
const PIN = {latitude: 48.1, longitude: 11.6}
const png = () => new Blob(["png"], {type: "image/png"})

function setup(linked = true) {
    const flags = new FakeFlagsBackend(async () => ({...PICTURE, url: "blob:painted"}))
    const handlers: FlagHandlers = {onOpened: vi.fn(), onChanged: vi.fn()}
    flags.listenForFlags(handlers)
    const submissions = new FakeFlagSubmissions(flags, async () => linked, async () => PICTURE, 0)
    return {flags, handlers, submissions}
}

const refusalOf = (sent: Promise<unknown>) =>
    sent.then(() => "sent", (e: unknown) => e instanceof SubmissionRefusedError ? e.refusal : e)

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    vi.useRealTimers()
})

describe("FakeFlagSubmissions", () => {
    it("keeps a submission waiting, refuses another while it waits, and lists the newest first", async () => {
        const {submissions} = setup()
        const upload = await submissions.uploadPicture(png())

        const sent = await submissions.submitFlag({name: " Bavaria ", uploadId: upload, pin: PIN})
        expect(sent).toMatchObject({name: "Bavaria", state: "waiting", change: false, flag: ""})
        expect(await refusalOf(submissions.submitFlag({name: "Tyrol", uploadId: await submissions.uploadPicture(png()), pin: PIN})))
            .toBe("waiting")
        expect((await submissions.mySubmissions()).map(({name}) => name)).toEqual(["Bavaria"])
    })

    it("refuses a guest, and lists it none", async () => {
        const {submissions} = setup(false)
        expect(await refusalOf(submissions.uploadPicture(png()))).toBe("guest")
        expect(await submissions.mySubmissions()).toEqual([])
    })

    it("approves, opens a flag with the upload's picture and the pin, and says the stream opened after each", async () => {
        const {submissions, handlers, flags} = setup()
        await submissions.submitFlag({name: "Bavaria", uploadId: await submissions.uploadPicture(png()), pin: PIN})
        vi.mocked(handlers.onOpened).mockClear()

        await submissions.approve()
        expect((await submissions.mySubmissions())[0]).toMatchObject({state: "approved", picture: PICTURE})
        expect(handlers.onOpened).toHaveBeenCalledTimes(1)

        await submissions.open(7)
        const [opened] = await submissions.mySubmissions()
        expect(opened.state).toBe("opened")
        const listed = (await flags.listFlags()).find(({code}) => code === opened.flag)
        expect(listed?.sold).toMatchObject({picture: PICTURE, pin: PIN})
        expect(handlers.onOpened).toHaveBeenCalledTimes(2)
    })

    it("rejects with a reason and the operator's note", async () => {
        const {submissions} = setup()
        await submissions.submitFlag({name: "FC Bayern", uploadId: await submissions.uploadPicture(png()), pin: PIN})

        submissions.reject("not-a-place", "A club's crest.")

        expect((await submissions.mySubmissions())[0]).toMatchObject({state: "rejected", rejection: {reason: "not-a-place", note: "A club's crest."}})
    })

    it("takes a change of a flag the account founded, with no new picture, and puts it on the flag once approved", async () => {
        const {submissions, flags} = setup()
        await submissions.submitFlag({name: "Bavaria", uploadId: await submissions.uploadPicture(png()), pin: PIN})
        await submissions.approve()
        await submissions.open()
        const [{flag}] = await submissions.mySubmissions()

        expect(await refusalOf(submissions.submitFlag({name: "Bayern", uploadId: "", pin: PIN, flag: "r-0000000000"}))).toBe("not-yours")
        await submissions.submitFlag({name: "Bayern", uploadId: "", pin: PIN, flag})
        await submissions.approve()

        expect((await flags.listFlags()).find(({code}) => code === flag)?.name).toBe("Bayern")
    })
})
