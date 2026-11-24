// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {FlagSubmitter, SubmissionRefusedError, SubmittedFlag} from "../../backends/flags.ts"
import {UploadOptions} from "../../backends/support.ts"
import {FlagCatalog, FlagPicture} from "../../domain/flags.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {useScreen} from "../hud/screenStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagSubmissionForm from "./FlagSubmissionForm.tsx"
import {useFlags} from "./flagsStore.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"

const PIN = {latitude: 48.137, longitude: 11.575}
const BAVARIA = "r-k3p9x2ma0b"
const PICTURE: FlagPicture = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}

const waiting = (sent: SubmittedFlag): Submission => ({
    id: "s-1", name: sent.name, pin: sent.pin, change: sent.flag !== undefined, flag: sent.flag ?? "", state: "waiting",
    submittedAt: 1, decidedAt: 1,
})

// An upload the test lands by hand, step by step.
function bucket() {
    const uploads: {options?: UploadOptions, land: (id: string) => void, fail: () => void}[] = []
    const uploadPicture = vi.fn((_file: Blob, options?: UploadOptions) => new Promise<string>((resolve, reject) => {
        uploads.push({options, land: resolve, fail: () => reject(new Error("the bucket hung up"))})
    }))
    const submitFlag = vi.fn(async (sent: SubmittedFlag) => waiting(sent))
    const submitter: FlagSubmitter = {uploadPicture, submitFlag, mySubmissions: vi.fn(async () => [])}
    return {submitter, uploads, uploadPicture, submitFlag}
}

const png = (name = "bavaria.png") => new File(["png"], name, {type: "image/png"})
const sendButton = () => screen.getByRole("button", {name: /^(Send|Uploading|Sending)/}) as HTMLButtonElement
const nameField = () => screen.getByRole("textbox") as HTMLInputElement
const fileInput = () => document.querySelector("input[type=file]") as HTMLInputElement

beforeEach(() => {
    URL.createObjectURL = vi.fn(() => "blob:preview")
    URL.revokeObjectURL = vi.fn()
    useFlagSubmissions.getState().serve({uploadPicture: vi.fn(), submitFlag: vi.fn(), mySubmissions: vi.fn(async () => [])})
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.getState().serve(undefined)
    useFlags.setState({catalog: FlagCatalog.BUILT_IN})
    useScreen.setState({sheet: undefined})
})

describe("FlagSubmissionForm", () => {
    it("uploads the picture with a bar, then sends the name, the upload and the pin, under StrictMode", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        useFlagForm.getState().compose({name: "  Bavaria ", file: png(), pinned: {pin: PIN, place: "Upper Bavaria"}})
        render(<StrictMode><FlagSubmissionForm submitter={submitter}/></StrictMode>)
        const user = userEvent.setup()

        await user.click(sendButton())
        expect(uploads[0].options?.signal?.aborted).toBe(false)
        act(() => uploads[0].options?.onProgress?.(0.5))
        expect(sendButton().textContent).toBe("Uploading… 50%")
        expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("50")

        await act(async () => uploads[0].land("u-1"))

        expect(submitFlag).toHaveBeenCalledWith({name: "Bavaria", uploadId: "u-1", pin: PIN, flag: undefined})
        expect(useFlagSubmissions.getState().submissions.map(({state}) => state)).toEqual(["waiting"])
        expect(useFlagForm.getState().composing).toBe(false)
    })

    it("waits for a name, a picture and a pin, and counts the name", async () => {
        const {submitter} = bucket()
        useFlagForm.getState().compose()
        render(<FlagSubmissionForm submitter={submitter}/>)
        const user = userEvent.setup({applyAccept: false})

        expect(sendButton().disabled).toBe(true)
        await user.type(nameField(), "Bavaria")
        expect(screen.getByText("7/40")).not.toBeNull()
        await user.upload(fileInput(), png())
        expect(sendButton().disabled).toBe(true)

        act(() => useFlagForm.setState({draft: {...useFlagForm.getState().draft, pinned: {pin: PIN}}}))
        expect(screen.getByText("48.14° N, 11.58° E")).not.toBeNull()
        expect(sendButton().disabled).toBe(false)
    })

    it("refuses a picture that is not a PNG, a JPEG or a WebP, and keeps none", async () => {
        const {submitter} = bucket()
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionForm submitter={submitter}/>)
        const user = userEvent.setup({applyAccept: false})

        await user.upload(fileInput(), new File(["gif"], "wave.gif", {type: "image/gif"}))

        expect(screen.getByRole("alert").textContent).toBe("wave.gif is not a PNG, JPEG or WebP picture.")
        expect(useFlagForm.getState().draft.file).toBeUndefined()
    })

    it("puts the form aside to aim the pin, closing the phone's sheet", async () => {
        const {submitter} = bucket()
        useScreen.setState({sheet: "board"})
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionForm submitter={submitter}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Pin it on the globe"}))

        expect(useFlagForm.getState().aiming).toBe(true)
        expect(useScreen.getState().sheet).toBeUndefined()
    })

    it("says a refusal in one line and keeps what was typed", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        submitFlag.mockRejectedValue(new SubmissionRefusedError("waiting"))
        const file = png()
        useFlagForm.getState().compose({name: "Bavaria", file, pinned: {pin: PIN}})
        render(<FlagSubmissionForm submitter={submitter}/>)
        const user = userEvent.setup()

        await user.click(sendButton())
        await act(async () => uploads[0].land("u-1"))

        expect(screen.getByRole("alert").textContent).toBe("One of your flags is waiting already.")
        expect(useFlagForm.getState().draft).toEqual({name: "Bavaria", file, pinned: {pin: PIN}})
        expect(nameField().value).toBe("Bavaria")
        expect(sendButton().disabled).toBe(false)
    })

    it("says a picture that did not go up, and sends nothing", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        vi.spyOn(console, "error").mockImplementation(() => {})
        useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
        render(<FlagSubmissionForm submitter={submitter}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].fail())

        expect(screen.getByRole("alert").textContent).toBe("The picture did not upload. Try again.")
        expect(submitFlag).not.toHaveBeenCalled()
    })

    it("sends a change for a flag the account founded with its picture", async () => {
        const {submitter, uploads, uploadPicture, submitFlag} = bucket()
        useFlags.setState({catalog: FlagCatalog.BUILT_IN.withFlags([{code: BAVARIA, name: "Bavaria", closed: false, sold: {picture: PICTURE, endsAt: 0, pin: PIN}}])})
        useFlagForm.getState().compose({name: "Bavaria", pinned: {pin: PIN}, flag: BAVARIA})
        const view = render(<FlagSubmissionForm submitter={submitter}/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Change your flag"})).not.toBeNull()
        await user.clear(nameField())
        await user.type(nameField(), "Bayern")
        expect(sendButton().disabled).toBe(true)

        view.unmount()
        useFlagForm.getState().compose({name: "Bayern", file: png(), pinned: {pin: PIN}, flag: BAVARIA})
        render(<FlagSubmissionForm submitter={submitter}/>)
        await user.click(sendButton())
        await act(async () => uploads[0].land("upload-1"))

        expect(uploadPicture).toHaveBeenCalledOnce()
        expect(submitFlag).toHaveBeenCalledWith({name: "Bayern", uploadId: "upload-1", pin: PIN, flag: BAVARIA})
    })

    it("calls the upload off when it closes", async () => {
        const {submitter, uploads} = bucket()
        useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
        const view = render(<FlagSubmissionForm submitter={submitter}/>)

        await userEvent.setup().click(sendButton())
        view.unmount()

        expect(uploads[0].options?.signal?.aborted).toBe(true)
    })
})
