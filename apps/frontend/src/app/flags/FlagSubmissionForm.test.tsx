// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {FlagSubmitter, SubmissionRefusedError, SubmittedFlag} from "../../backends/flags.ts"
import {Shop, ShopRefusedError} from "../../backends/shop.ts"
import {UploadOptions} from "../../backends/support.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import type {PlanPrice} from "../../domain/shop.ts"
import {useScreen} from "../hud/screenStore.ts"
import {useShop} from "../shop/shopStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagSubmissionForm from "./FlagSubmissionForm.tsx"
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
    const submitter: FlagSubmitter = {uploadPicture, submitFlag, mySubmissions: vi.fn(async () => [])}
    return {submitter, uploads, uploadPicture, submitFlag}
}

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

const shopOf = (checkout: Shop["checkout"] = vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1")): Shop => ({
    plans: vi.fn(async () => PLANS),
    checkout,
    myRenewals: vi.fn(async () => []),
    stopRenewal: vi.fn(),
})

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
    useScreen.setState({sheet: undefined})
    useShop.setState(useShop.getInitialState(), true)
})

describe("FlagSubmissionForm", () => {
    it("uploads the picture with a bar, then sends the name, the upload and the pin, under StrictMode", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        useFlagForm.getState().compose({name: "  Bavaria ", file: png(), pinned: {pin: PIN, place: "Upper Bavaria"}})
        render(<StrictMode><FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/></StrictMode>)
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
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)
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
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)
        const user = userEvent.setup({applyAccept: false})

        await user.upload(fileInput(), new File(["gif"], "wave.gif", {type: "image/gif"}))

        expect(screen.getByRole("alert").textContent).toBe("wave.gif is not a PNG, JPEG or WebP picture.")
        expect(useFlagForm.getState().draft.file).toBeUndefined()
    })

    it("puts the form aside to aim the pin, closing the phone's sheet", async () => {
        const {submitter} = bucket()
        useScreen.setState({sheet: "board"})
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Put the capital on the globe"}))

        expect(useFlagForm.getState().aiming).toBe(true)
        expect(useScreen.getState().sheet).toBeUndefined()
    })

    it("says a refusal in one line and keeps what was typed", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        submitFlag.mockRejectedValue(new SubmissionRefusedError("waiting"))
        const file = png()
        useFlagForm.getState().compose({name: "Bavaria", file, pinned: {pin: PIN}})
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)
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
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].fail())

        expect(screen.getByRole("alert").textContent).toBe("The picture did not upload. Try again.")
        expect(submitFlag).not.toHaveBeenCalled()
    })

    it("sends a change of the submission that waits with a picture picked again", async () => {
        const {submitter, uploads, uploadPicture, submitFlag} = bucket()
        useFlagForm.getState().compose({name: "Bavaria", pinned: {pin: PIN}, replaces: "s-1"})
        const view = render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Change your flag"})).not.toBeNull()
        await user.clear(nameField())
        await user.type(nameField(), "Bayern")
        expect(sendButton().disabled).toBe(true)

        view.unmount()
        useFlagForm.getState().compose({name: "Bayern", file: png(), pinned: {pin: PIN}, replaces: "s-1"})
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)
        await user.click(sendButton())
        await act(async () => uploads[0].land("upload-1"))

        expect(uploadPicture).toHaveBeenCalledOnce()
        expect(submitFlag).toHaveBeenCalledWith({name: "Bayern", uploadId: "upload-1", pin: PIN, replaces: "s-1"})
    })

    it("says the operator decided the submission it was changing", async () => {
        const {submitter, uploads, submitFlag} = bucket()
        submitFlag.mockRejectedValueOnce(new SubmissionRefusedError("decided"))
        useFlagForm.getState().compose({name: "Bayern", file: png(), pinned: {pin: PIN}, replaces: "s-1"})
        render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)

        await userEvent.setup().click(sendButton())
        await act(async () => uploads[0].land("upload-1"))

        expect(screen.getByRole("alert").textContent).toBe("Your flag was reviewed already, so it can't change.")
    })

    it("calls the upload off when it closes", async () => {
        const {submitter, uploads} = bucket()
        useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
        const view = render(<FlagSubmissionForm submitter={submitter} shop={shopOf()} toCheckout={vi.fn()}/>)

        await userEvent.setup().click(sendButton())
        view.unmount()

        expect(uploads[0].options?.signal?.aborted).toBe(true)
    })

    describe("while the shop sells", () => {
        const payButton = (name: RegExp | string) => screen.getByRole("button", {name}) as HTMLButtonElement

        it("ends with the plans and the consent, and pays for the flag as it sends it", async () => {
            const {submitter, uploads, submitFlag} = bucket()
            const checkout = vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1")
            const toCheckout = vi.fn()
            useShop.setState({plans: PLANS})
            useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
            render(<FlagSubmissionForm submitter={submitter} shop={shopOf(checkout)} toCheckout={toCheckout}/>)
            const user = userEvent.setup()

            expect(screen.getByRole("radio", {name: /1 year/}).getAttribute("aria-checked")).toBe("true")
            expect(payButton("Pay €60").disabled).toBe(true)
            await user.click(screen.getByRole("checkbox", {name: "I want my flag to start as soon as it's approved. I know I can't cancel once it starts."}))
            await user.click(payButton("Pay €60"))
            await act(async () => uploads[0].land("u-1"))

            expect(submitFlag).toHaveBeenCalledOnce()
            expect(checkout).toHaveBeenCalledWith({plan: "year", target: {kind: "submission", id: "s-2"}, startsNow: true})
            expect(toCheckout).toHaveBeenCalledWith("https://checkout.stripe.com/c/pay/cs_1")
            expect(useFlagSubmissions.getState().submissions.map(({id}) => id)).toEqual(["s-2"])
        })

        it("pays for the plan picked in the shop", async () => {
            const {submitter, uploads} = bucket()
            const checkout = vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1")
            useShop.setState({plans: PLANS, chosen: "month"})
            useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
            render(<FlagSubmissionForm submitter={submitter} shop={shopOf(checkout)} toCheckout={vi.fn()}/>)
            const user = userEvent.setup()

            await user.click(screen.getByRole("checkbox"))
            await user.click(payButton("Pay €10"))
            await act(async () => uploads[0].land("u-1"))

            expect(checkout).toHaveBeenCalledWith({plan: "month", target: {kind: "submission", id: "s-2"}, startsNow: true})
        })

        it("keeps the flag sent when its payment page does not open, and says why in the place", async () => {
            const {submitter, uploads} = bucket()
            const toCheckout = vi.fn()
            useShop.setState({plans: PLANS})
            useFlagForm.getState().compose({name: "Bavaria", file: png(), pinned: {pin: PIN}})
            render(<FlagSubmissionForm submitter={submitter}
                                       shop={shopOf(vi.fn(async (): Promise<string> => {
                                           throw new ShopRefusedError("unavailable")
                                       }))}
                                       toCheckout={toCheckout}/>)
            const user = userEvent.setup()

            await user.click(screen.getByRole("checkbox"))
            await user.click(payButton("Pay €60"))
            await act(async () => uploads[0].land("u-1"))

            expect(toCheckout).not.toHaveBeenCalled()
            expect(useFlagSubmissions.getState().submissions.map(({id}) => id)).toEqual(["s-2"])
            expect(useFlagForm.getState()).toMatchObject({composing: false, payFailure: "unavailable"})
        })
    })
})
