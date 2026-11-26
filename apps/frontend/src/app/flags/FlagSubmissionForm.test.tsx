// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {FlagSubmitter, SubmissionRefusedError, SubmittedFlag} from "../../backends/flags.ts"
import {UploadOptions} from "../../backends/support.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import type {PlanPrice} from "../../domain/shop.ts"
import {useScreen} from "../hud/screenStore.ts"
import {useShop} from "../shop/shopStore.ts"
import {useFlags} from "./flagsStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagSubmissionForm from "./FlagSubmissionForm.tsx"
import {FlagShooter} from "./mapShot.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"

const PIN = {latitude: 48.137, longitude: 11.575}

const waiting = (sent: SubmittedFlag): Submission => ({
    id: "s-2", name: sent.name, pin: sent.pin, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false,
})

// An upload the test lands by hand, step by step.
function bucket() {
    const uploads: {options?: UploadOptions, land: (id: string) => void, fail: () => void}[] = []
    const uploadPicture = vi.fn((_file: Blob, options?: UploadOptions) => new Promise<string>((resolve, reject) => {
        uploads.push({options, land: resolve, fail: () => reject(new Error("the bucket hung up"))})
    }))
    const submitFlag = vi.fn(async (sent: SubmittedFlag) => waiting(sent))
    const submitter: FlagSubmitter = {uploadPicture, submitFlag, myFlags: vi.fn(async () => ({submissions: [], held: []}))}
    return {submitter, uploads, uploadPicture, submitFlag}
}

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

// The globe is in no test: its shot never comes.
const NEVER_SHOT: FlagShooter = () => new Promise(() => {})

const png = (name = "bavaria.png") => new File(["png"], name, {type: "image/png"})
const sendButton = () => screen.getByRole("button", {name: /^(Next|Uploading|Sending)/}) as HTMLButtonElement
const nameField = () => screen.getByRole("textbox") as HTMLInputElement
const fileInput = () => document.querySelector("input[type=file]") as HTMLInputElement

beforeEach(() => {
    URL.createObjectURL = vi.fn(() => "blob:preview")
    URL.revokeObjectURL = vi.fn()
    useFlagSubmissions.getState().serve({uploadPicture: vi.fn(), submitFlag: vi.fn(), myFlags: vi.fn(async () => ({submissions: [], held: []}))})
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.getState().serve(undefined)
    useScreen.setState({sheet: undefined})
    useShop.setState(useShop.getInitialState(), true)
})

describe("FlagSubmissionForm", () => {
    it("uploads the picture with a bar, then sends the name, the upload and the pin, under StrictMode", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        useFlagForm.getState().compose({name: "  Bavaria ", file: png(), pinned: {pin: PIN, place: "Upper Bavaria"}})
        render(<StrictMode><FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/></StrictMode>)
        const user = userEvent.setup()

        await user.click(sendButton())
        expect(uploads[0].options?.signal?.aborted).toBe(false)
        act(() => uploads[0].options?.onProgress?.(0.5))
        expect(sendButton().textContent).toBe("Uploading… 50%")
        expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("50")

        await act(async () => uploads[0].land("u-1"))

        expect(submitFlag).toHaveBeenCalledWith({name: "Bavaria", uploadId: "u-1", pin: PIN, flag: undefined})
        expect(useFlagSubmissions.getState().submissions.map(({state}) => state)).toEqual(["waiting"])
    })

    it("goes to the paywall once sent, keeping what was typed for Back and Fix it", async () => {
        const {submitter, uploads} = bucket()
        const draft = {name: "Bavaria", file: png(), pinned: {pin: PIN, place: "Upper Bavaria"}}
        useFlagForm.getState().compose(draft)
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].land("u-1"))

        expect(useFlagForm.getState()).toMatchObject({
            step: {kind: "pay", submission: {id: "s-2", name: "Bavaria"}, typed: draft},
            draft,
        })
    })

    it("shows no price, even while the shop sells", () => {
        useShop.setState({plans: PLANS})
        useFlagForm.getState().compose()
        render(<FlagSubmissionForm submitter={bucket().submitter} shooter={NEVER_SHOT}/>)

        expect(screen.queryByRole("radio")).toBeNull()
        expect(screen.queryByText(/€/)).toBeNull()
        expect(sendButton().textContent).toBe("Next")
    })

    it("shows the flag as the players will see it: on the map at its capital, its name and the capital's place", async () => {
        const file = png()
        const shooter = vi.fn(async () => ({url: "blob:map", width: 560, height: 280, capital: {x: 4, y: -2}}))
        useFlagForm.getState().compose({name: "Bavaria", file, pinned: {pin: PIN, place: "Upper Bavaria"}})
        render(<FlagSubmissionForm submitter={bucket().submitter} shooter={shooter}/>)

        const card = document.querySelector(".flag-card") as HTMLElement
        expect(card.querySelector("img")?.getAttribute("src"), "the picture alone until the map is shot").toBe("blob:preview")
        expect(card.textContent).toBe("BavariaUpper Bavaria")

        const shot = await vi.waitFor(() => {
            const shown = card.querySelector<HTMLImageElement>(".flag-card-shot")
            expect(shown).not.toBeNull()
            return shown!
        })
        expect(shooter).toHaveBeenCalledWith(PIN, file)
        expect(shot.getAttribute("src")).toBe("blob:map")
        expect(shot.style.width).toBe("560px")
        expect(card.querySelector<HTMLElement>(".flag-card-crown")?.style.left).toBe("calc(50% + 4px)")
    })

    it("shoots no map before the capital is put", () => {
        const shooter = vi.fn(NEVER_SHOT)
        useFlagForm.getState().compose({name: "Bavaria", file: png()})
        render(<FlagSubmissionForm submitter={bucket().submitter} shooter={shooter}/>)

        expect(shooter).not.toHaveBeenCalled()
    })

    it("sells the region's flag over a new one, never over a change", () => {
        useFlagForm.getState().compose()
        const view = render(<FlagSubmissionForm submitter={bucket().submitter} shooter={NEVER_SHOT}/>)
        expect(screen.getByText("Put your region on the map")).not.toBeNull()
        view.unmount()

        useFlagForm.getState().compose({name: "Bavaria", replaces: "s-1"})
        render(<FlagSubmissionForm submitter={bucket().submitter} shooter={NEVER_SHOT}/>)
        expect(screen.queryByText("Put your region on the map")).toBeNull()
    })

    it("waits for a name, a picture and a pin, and counts the name", async () => {
        const {submitter} = bucket()
        useFlagForm.getState().compose()
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)
        const user = userEvent.setup({applyAccept: false})

        expect(sendButton().disabled).toBe(true)
        await user.type(nameField(), "Bavaria")
        expect(screen.getByText("7/40")).not.toBeNull()
        await user.upload(fileInput(), png())
        expect(sendButton().disabled).toBe(true)

        act(() => useFlagForm.setState({draft: {...useFlagForm.getState().draft, pinned: {pin: PIN}}}))
        expect(screen.getAllByText("48.14° N, 11.58° E")).toHaveLength(2)
        expect(sendButton().disabled).toBe(false)
    })

    it("refuses a picture that is not a PNG, a JPEG or a WebP, and keeps none", async () => {
        const {submitter} = bucket()
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)
        const user = userEvent.setup({applyAccept: false})

        await user.upload(fileInput(), new File(["gif"], "wave.gif", {type: "image/gif"}))

        expect(screen.getByRole("alert").textContent).toBe("wave.gif is not a PNG, JPEG or WebP picture.")
        expect(useFlagForm.getState().draft.file).toBeUndefined()
    })

    it("puts the form aside to aim the pin, closing the phone's sheet", async () => {
        const {submitter} = bucket()
        useScreen.setState({sheet: "board"})
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Put the capital on the globe"}))

        expect(useFlagForm.getState().aiming).toBe(true)
        expect(useScreen.getState().sheet).toBeUndefined()
    })

    it("says a refusal in one line and keeps what was typed", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        submitFlag.mockRejectedValue(new SubmissionRefusedError("waiting"))
        const file = png()
        useFlagForm.getState().compose({name: "Bavaria", file, pinned: {pin: PIN}})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)
        const user = userEvent.setup()

        await user.click(sendButton())
        await act(async () => uploads[0].land("u-1"))

        expect(screen.getByRole("alert").textContent).toBe("One of your flags is waiting already.")
        expect(useFlagForm.getState().draft).toEqual({name: "Bavaria", file, pinned: {pin: PIN}})
        expect(nameField().value).toBe("Bavaria")
        expect(sendButton().disabled).toBe(false)
    })

    it("takes a name of a flag whose time ended to its claim, never to a new flag", async () => {
        useFlags.setState({catalog: FlagCatalog.BUILT_IN.withFlags([{
            code: "r-k3p9x2ma0b", name: "Bavaria", closed: true,
            sold: {picture: {url: "https://media.test/b.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}, endsAt: 1, closedAt: 1},
        }])})
        useFlagForm.getState().compose({name: "bavaria ", file: png(), pinned: {pin: PIN}})
        render(<FlagSubmissionForm submitter={bucket().submitter} shooter={NEVER_SHOT}/>)

        expect(screen.getByRole("status").textContent).toBe("Bavaria is free. Claim it.Claim it")
        expect(sendButton().disabled).toBe(true)
        await userEvent.setup().click(screen.getByRole("button", {name: "Claim it"}))

        expect(useFlagForm.getState().step).toEqual({kind: "flag", flag: "r-k3p9x2ma0b", fromHub: false})
        useFlags.setState({catalog: FlagCatalog.BUILT_IN})
    })

    it("sends a new flag in the place of the one that waits unpaid, and alone once that one was reviewed", async () => {
        const {submitter, uploads, uploadPicture, submitFlag} = bucket()
        submitFlag.mockRejectedValueOnce(new SubmissionRefusedError("decided"))
        useFlagForm.getState().compose({name: "Swabia", file: png(), pinned: {pin: PIN}, replaces: "s-1", replacing: "Sardinia"})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Add your region's flag"})).not.toBeNull()
        expect(screen.getByText("This replaces Sardinia, which waits.")).not.toBeNull()
        await user.click(sendButton())
        await act(async () => uploads[0].land("u-1"))
        await act(async () => uploads[1].land("u-2"))

        expect(uploadPicture).toHaveBeenCalledTimes(2)
        expect(submitFlag.mock.calls.map(([sent]) => [sent.uploadId, sent.replaces])).toEqual([["u-1", "s-1"], ["u-2", undefined]])
        expect(useFlagForm.getState().step).toMatchObject({kind: "pay", submission: {name: "Swabia"}})
    })

    it("says a picture that did not go up, and sends nothing", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        vi.spyOn(console, "error").mockImplementation(() => {})
        useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].fail())

        expect(screen.getByRole("alert").textContent).toBe("The picture did not upload. Try again.")
        expect(submitFlag).not.toHaveBeenCalled()
    })

    it("sends a change of the submission that waits with a picture picked again", async () => {
        const {submitter, uploads, uploadPicture, submitFlag} = bucket()
        useFlagForm.getState().compose({name: "Bavaria", pinned: {pin: PIN}, replaces: "s-1"})
        const view = render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Change your flag"})).not.toBeNull()
        await user.clear(nameField())
        await user.type(nameField(), "Bayern")
        expect(sendButton().disabled).toBe(true)

        view.unmount()
        useFlagForm.getState().compose({name: "Bayern", file: png(), pinned: {pin: PIN}, replaces: "s-1"})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)
        await user.click(sendButton())
        await act(async () => uploads[0].land("upload-1"))

        expect(uploadPicture).toHaveBeenCalledOnce()
        expect(submitFlag).toHaveBeenCalledWith({name: "Bayern", uploadId: "upload-1", pin: PIN, replaces: "s-1"})
    })

    it("shows the verdict of the submission it was changing, reviewed meanwhile, on its paywall", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        submitFlag.mockRejectedValueOnce(new SubmissionRefusedError("decided"))
        const sent: Submission = {id: "s-1", name: "Bavaria", pin: PIN, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false}
        act(() => useFlagSubmissions.setState({submissions: [sent], read: true}))
        useFlagForm.getState().compose({name: "Bayern", file: png(), pinned: {pin: PIN}, replaces: "s-1"})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].land("upload-1"))

        expect(useFlagForm.getState().step).toEqual({kind: "pay", submission: sent, typed: undefined})
        expect(useFlagForm.getState().draft).toMatchObject({name: "Bayern", replaces: undefined})
    })

    it("says the operator decided a submission it no longer knows", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        submitFlag.mockRejectedValueOnce(new SubmissionRefusedError("decided"))
        useFlagForm.getState().compose({name: "Bayern", file: png(), pinned: {pin: PIN}, replaces: "s-1"})
        render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].land("upload-1"))

        expect(screen.getByRole("alert").textContent).toBe("Your flag was reviewed already, so it can't change.")
    })

    it("calls the upload off when it closes", async () => {
        const {submitter, uploads} = bucket()
        useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
        const view = render(<FlagSubmissionForm submitter={submitter} shooter={NEVER_SHOT}/>)

        await userEvent.setup().click(sendButton())
        view.unmount()

        expect(uploads[0].options?.signal?.aborted).toBe(true)
    })
})
