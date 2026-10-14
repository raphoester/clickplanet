// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AttachmentRules, NO_ATTACHMENTS, PageContext} from "../../backends/support.ts"
import FeedbackModal from "./FeedbackModal.tsx"

const PAGE: PageContext = {
    build: "play-abc.js",
    viewportWidth: 390,
    viewportHeight: 844,
    pixelRatio: 3,
    connectionDown: false,
    gpuVendor: "Apple Inc.",
    gpuRenderer: "Apple GPU",
    webglVersion: "WebGL 2.0",
    maxTextureSize: 4096,
    contextLost: false,
    cores: 6,
    memoryGb: 4,
}

beforeEach(() => {
    // jsdom draws nothing, so it makes no object URL.
    URL.createObjectURL = vi.fn(() => "blob:thumb")
    URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

const TAKES_FILES: AttachmentRules = {maxPerNote: 10, maxBytes: 1000, contentTypes: ["image/png", "video/mp4"]}

function setup(rules: AttachmentRules = NO_ATTACHMENTS) {
    const uploaded: Blob[] = []
    const support = {
        suggestFeature: vi.fn(async () => undefined),
        reportBug: vi.fn(async () => undefined),
        attachmentRules: vi.fn(async () => rules),
        uploadAttachment: vi.fn(async (file: Blob) => `file-${uploaded.push(file) - 1}`),
    }
    const onClose = vi.fn()
    render(<FeedbackModal support={support} pageContext={() => PAGE} onClose={onClose}/>)
    return {support, onClose, user: userEvent.setup()}
}

const kind = (name: string) => screen.getByRole("radio", {name}) as HTMLButtonElement
const box = (name: string) => screen.getByRole("textbox", {name}) as HTMLTextAreaElement
const send = () => screen.getByRole("button", {name: "Send"}) as HTMLButtonElement

describe("FeedbackModal", () => {
    it("asks whether it is an idea or a bug first, and sends nothing yet", () => {
        setup()

        expect(screen.getByRole("dialog", {name: "Send feedback"})).not.toBeNull()
        expect(screen.getAllByRole("radio").map((option) => option.textContent)).toEqual(["An idea", "A bug"])
        expect(document.activeElement).toBe(kind("An idea"))
        expect(send().disabled).toBe(true)
    })

    it("waits for the kind and for the text", async () => {
        const {user} = setup()

        await user.type(box("Your message"), "a night mode")
        expect(send().disabled).toBe(true)

        await user.click(kind("An idea"))
        expect(send().disabled).toBe(false)

        await user.clear(box("Your idea"))
        expect(send().disabled).toBe(true)
    })

    it("sends an idea as a suggestion, with what the page knows", async () => {
        const {support, user} = setup()

        await user.click(kind("An idea"))
        await user.type(box("Your idea"), "a night mode")
        await user.click(send())

        expect(support.suggestFeature).toHaveBeenCalledWith("a night mode", PAGE, [])
        expect(support.reportBug).not.toHaveBeenCalled()
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
    })

    it("asks what happened for a bug, and sends it as a bug report", async () => {
        const {support, user} = setup()

        await user.click(kind("A bug"))
        expect(box("What happened").placeholder).toBe("What did you do, and what went wrong?")
        await user.type(box("What happened"), "the globe is black")
        await user.click(send())

        expect(support.reportBug).toHaveBeenCalledWith("the globe is black", PAGE, [])
        expect(support.suggestFeature).not.toHaveBeenCalled()
    })

    it("keeps the text when the kind changes", async () => {
        const {support, user} = setup()

        await user.click(kind("An idea"))
        await user.type(box("Your idea"), "the globe is black")
        await user.click(kind("A bug"))

        expect(kind("A bug").getAttribute("aria-checked")).toBe("true")
        expect(kind("An idea").getAttribute("aria-checked")).toBe("false")
        expect(box("What happened").value).toBe("the globe is black")

        await user.click(send())
        expect(support.reportBug).toHaveBeenCalledTimes(1)
    })

    it("offers no file on a server that takes none", async () => {
        const {support} = setup()

        await waitFor(() => expect(support.attachmentRules).toHaveBeenCalledTimes(1))

        expect(screen.queryByRole("button", {name: "Add photos or videos"})).toBeNull()
    })

    it.each([
        ["A bug", "What happened", "reportBug"],
        ["An idea", "Your idea", "suggestFeature"],
    ] as const)("uploads the files picked for %s, then sends it with them", async (option, label, procedure) => {
        const {support, user} = setup(TAKES_FILES)
        const screenshot = new File(["a screenshot"], "screen.png", {type: "image/png"})
        const recording = new File(["a screen recording"], "clip.mp4", {type: "video/mp4"})

        await screen.findByRole("button", {name: "Add photos or videos"})
        await user.upload(document.querySelector("input[type=file]") as HTMLInputElement, [screenshot, recording])
        await user.click(kind(option))
        await user.type(box(label), "the globe is black")
        await user.click(send())

        await waitFor(() => expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!"))
        expect(support.uploadAttachment.mock.calls.map((call) => call[0])).toEqual([screenshot, recording])
        expect(support[procedure]).toHaveBeenCalledWith("the globe is black", PAGE, ["file-0", "file-1"])
    })
})
