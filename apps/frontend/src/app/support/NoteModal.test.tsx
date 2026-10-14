// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {
    AttachmentRules,
    MAX_NOTE_LENGTH,
    NO_ATTACHMENTS,
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    UploadOptions,
} from "../../backends/support.ts"
import NoteModal, {NoteFiles} from "./NoteModal.tsx"

beforeEach(() => {
    // jsdom draws nothing, so it makes no object URL.
    URL.createObjectURL = vi.fn(() => "blob:thumb")
    URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

type Send = (text: string, attachmentIds: string[]) => Promise<void>

function setup(onSend: Send = vi.fn(async () => undefined), files?: NoteFiles) {
    const onClose = vi.fn()
    render(<NoteModal title="Suggest a feature"
                      label="Your idea"
                      placeholder="What should the game have?"
                      files={files}
                      onSend={onSend}
                      onClose={onClose}/>)
    return {onSend, onClose, user: userEvent.setup({applyAccept: false})}
}

const box = () => screen.getByRole("textbox", {name: "Your idea"}) as HTMLTextAreaElement
const send = () => screen.getByRole("button", {name: "Send"}) as HTMLButtonElement

describe("NoteModal", () => {
    it("opens on its text box, with nothing to send yet", () => {
        setup()

        expect(screen.getByRole("dialog", {name: "Suggest a feature"})).not.toBeNull()
        expect(document.activeElement).toBe(box())
        expect(send().disabled).toBe(true)
    })

    it("sends the text trimmed, lines kept, and says it was sent", async () => {
        const {onSend, onClose, user} = setup()

        await user.type(box(), "  a night mode{Enter}for the globe  ")
        await user.click(send())

        expect(onSend).toHaveBeenCalledWith("a night mode\nfor the globe", [])
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
        expect(screen.queryByRole("textbox")).toBeNull()

        await user.click(screen.getAllByRole("button", {name: "Close"}).at(-1)!)
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("sends nothing while one send is in flight", async () => {
        let land = () => {}
        const onSend = vi.fn(() => new Promise<void>((resolve) => {
            land = () => resolve()
        }))
        const {user} = setup(onSend)

        await user.type(box(), "a night mode")
        await user.click(send())

        expect(send().disabled).toBe(true)
        expect(box().readOnly).toBe(true)
        await user.click(send())
        expect(onSend).toHaveBeenCalledTimes(1)

        land()
        expect((await screen.findByRole("status")).textContent).toBe("Sent. Thank you!")
    })

    it.each([
        [new SupportRateLimitedError(), "You sent too many. Try again later."],
        [new SupportRejectedError(), "That was refused."],
        [new SupportNoSessionError(), "Could not start a session. Try again in a moment."],
    ])("says why %s refused it, and keeps the text", async (refusal, notice) => {
        const {user} = setup(vi.fn(async () => {
            throw refusal
        }))

        await user.type(box(), "a night mode")
        await user.click(send())

        expect(screen.getByRole("alert").textContent).toBe(notice)
        expect(box().value).toBe("a night mode")
        expect(send().disabled).toBe(false)
    })

    it("says a failure it does not know could not be sent, and logs it", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {user} = setup(vi.fn(async () => {
            throw new Error("the connection dropped")
        }))

        await user.type(box(), "a night mode")
        await user.click(send())

        expect(screen.getByRole("alert").textContent).toBe("It could not be sent. Try again.")
        expect(logged).toHaveBeenCalledTimes(1)
    })

    it("counts down near the limit, and refuses a text past it", async () => {
        const {user} = setup()

        await user.click(box())
        await user.paste("a".repeat(MAX_NOTE_LENGTH - 5))
        expect(screen.getByText("5")).not.toBeNull()
        expect(send().disabled).toBe(false)

        await user.paste("🌍".repeat(6))
        expect(screen.getByText("-1")).not.toBeNull()
        expect(send().disabled).toBe(true)
    })

    it("does not close on a click beside it once something is typed", async () => {
        const {onClose, user} = setup()
        const backdrop = document.querySelector(".modal")!

        await user.click(backdrop)
        expect(onClose).toHaveBeenCalledTimes(1)

        await user.type(box(), "a night mode")
        await user.click(backdrop)
        expect(onClose).toHaveBeenCalledTimes(1)
    })
})

const rules: AttachmentRules = {maxPerNote: 3, maxBytes: 1000, contentTypes: ["image/png", "video/mp4"]}

const screenshot = new File(["a screenshot"], "screen.png", {type: "image/png"})
const recording = new File(["a screen recording"], "clip.mp4", {type: "video/mp4"})

type Landing = {
    file: Blob
    land: (id: string) => void
    fail: (e?: Error) => void
    progress: (fraction: number) => void
    signal?: AbortSignal
}

function bucket(read: AttachmentRules | Error = rules) {
    const landings: Landing[] = []
    const files = {
        attachmentRules: vi.fn(async () => {
            if (read instanceof Error) throw read
            return read
        }),
        uploadAttachment: vi.fn((file: Blob, options?: UploadOptions) => new Promise<string>((resolve, reject) => {
            landings.push({
                file,
                land: resolve,
                fail: (e = new Error("the upload was answered 403")) => reject(e),
                progress: (fraction) => options?.onProgress?.(fraction),
                signal: options?.signal,
            })
        })),
    }
    return {files, landings}
}

const addButton = () => screen.findByRole("button", {name: "Add photos or videos"}) as Promise<HTMLButtonElement>
const input = () => document.querySelector("input[type=file]") as HTMLInputElement
const attached = () => within(screen.getByRole("list", {name: "Attached files"})).getAllByRole("listitem")
const footer = () => screen.getByRole("button", {name: /^(Send|Uploading)/}) as HTMLButtonElement

async function writeWith(user: ReturnType<typeof userEvent.setup>, ...picked: File[]) {
    await addButton()
    await user.upload(input(), picked)
    await user.type(box(), "the globe is black")
}

describe("NoteModal's files", () => {
    it("takes a text alone when nothing says what a note may carry", () => {
        setup()

        expect(screen.queryByRole("button", {name: "Add photos or videos"})).toBeNull()
    })

    it("takes a text alone on a server that takes no file", async () => {
        const {files} = bucket(NO_ATTACHMENTS)
        setup(undefined, files)

        await act(async () => undefined)

        expect(screen.queryByRole("button", {name: "Add photos or videos"})).toBeNull()
    })

    it("takes a text alone when the rules cannot be read, and logs it", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {files} = bucket(new Error("the connection dropped"))
        setup(undefined, files)

        await act(async () => undefined)

        expect(screen.queryByRole("button", {name: "Add photos or videos"})).toBeNull()
        expect(logged).toHaveBeenCalledTimes(1)
    })

    it("offers the photos and the videos the server takes", async () => {
        const {files} = bucket()
        setup(undefined, files)

        await addButton()

        expect(input().accept).toBe("image/png,video/mp4")
        expect(input().multiple).toBe(true)
    })

    it("lists what was picked with its size, and takes one off", async () => {
        const {files} = bucket()
        const {user} = setup(undefined, files)

        await writeWith(user, screenshot, recording)

        expect(attached().map((item) => item.textContent)).toEqual(["screen.png12 B", "clip.mp418 B"])

        await user.click(screen.getByRole("button", {name: "Remove screen.png"}))

        expect(attached().map((item) => item.textContent)).toEqual(["clip.mp418 B"])
    })

    it.each([
        ["a file that is not a photo or a video", [new File(["x"], "notes.txt", {type: "text/plain"})], "notes.txt is not a photo or a video."],
        ["a file that is too big", [new File([new Uint8Array(1001)], "long.mp4", {type: "video/mp4"})], "long.mp4 is over 1000 B."],
        ["one file too many", [screenshot, screenshot, screenshot, screenshot], "3 files at most."],
    ])("refuses %s and says why", async (_, picked, notice) => {
        const {files} = bucket()
        const {user} = setup(undefined, files)

        await addButton()
        await user.upload(input(), picked)

        expect(screen.getByRole("alert").textContent).toBe(notice)
    })

    it("offers no more once the note is full", async () => {
        const {files} = bucket()
        const {user} = setup(undefined, files)

        await writeWith(user, screenshot, screenshot, screenshot)

        expect((await addButton()).disabled).toBe(true)
    })

    it("takes a picture pasted into the text", async () => {
        const {files} = bucket()
        setup(undefined, files)
        await addButton()

        fireEvent.paste(box(), {clipboardData: {files: [screenshot]}})

        expect(attached().map((item) => item.textContent)).toEqual(["screen.png12 B"])
    })

    it("leaves a pasted text to the box", async () => {
        const {files} = bucket()
        const {user} = setup(undefined, files)
        await addButton()

        await user.click(box())
        await user.paste("a night mode")

        expect(box().value).toBe("a night mode")
        expect(screen.queryByRole("list", {name: "Attached files"})).toBeNull()
    })

    it("uploads each file in turn, then sends the note with every one of them", async () => {
        const {files, landings} = bucket()
        const onSend = vi.fn(async () => undefined)
        const {user} = setup(onSend, files)

        await writeWith(user, screenshot, recording)
        await user.click(send())

        expect(files.uploadAttachment).toHaveBeenCalledTimes(1)
        expect(landings[0].file).toBe(screenshot)
        expect(footer().textContent).toBe("Uploading 1 of 2…")
        expect(footer().disabled).toBe(true)
        expect(box().readOnly).toBe(true)
        expect(onSend).not.toHaveBeenCalled()

        await act(async () => landings[0].progress(0.5))
        expect(screen.getByRole("progressbar", {name: "Uploading screen.png"}).getAttribute("aria-valuenow")).toBe("50")

        await act(async () => landings[0].land("first"))
        expect(landings[1].file).toBe(recording)
        expect(footer().textContent).toBe("Uploading 2 of 2…")
        expect(within(attached()[0]).getByRole("img", {name: "Uploaded"})).not.toBeNull()
        expect(onSend).not.toHaveBeenCalled()

        await act(async () => landings[1].land("second"))
        expect(onSend).toHaveBeenCalledWith("the globe is black", ["first", "second"])
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
    })

    it("says which file did not upload, sends nothing, and goes on from it", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {files, landings} = bucket()
        const onSend = vi.fn(async () => undefined)
        const {user} = setup(onSend, files)

        await writeWith(user, screenshot, recording)
        await user.click(send())
        await act(async () => landings[0].land("first"))
        await act(async () => landings[1].fail())

        expect(screen.getByRole("alert").textContent).toBe("clip.mp4 could not be uploaded. Try again.")
        expect(onSend).not.toHaveBeenCalled()
        expect(logged).toHaveBeenCalledTimes(1)
        expect(footer().disabled).toBe(false)

        await user.click(send())

        expect(files.uploadAttachment).toHaveBeenCalledTimes(3)
        expect(landings[2].file).toBe(recording)

        await act(async () => landings[2].land("second"))
        expect(onSend).toHaveBeenCalledWith("the globe is black", ["first", "second"])
    })

    it("says why the server would not take a file, and keeps them all", async () => {
        const {files, landings} = bucket()
        const onSend = vi.fn(async () => undefined)
        const {user} = setup(onSend, files)

        await writeWith(user, screenshot)
        await user.click(send())
        await act(async () => landings[0].fail(new SupportRateLimitedError()))

        expect(screen.getByRole("alert").textContent).toBe("You sent too many. Try again later.")
        expect(onSend).not.toHaveBeenCalled()
        expect(attached()).toHaveLength(1)
    })

    it("sends again a note that was refused with the files already up", async () => {
        const {files, landings} = bucket()
        const onSend = vi.fn()
            .mockRejectedValueOnce(new SupportRateLimitedError())
            .mockResolvedValueOnce(undefined)
        const {user} = setup(onSend, files)

        await writeWith(user, screenshot)
        await user.click(send())
        await act(async () => landings[0].land("first"))

        expect(screen.getByRole("alert").textContent).toBe("You sent too many. Try again later.")

        await user.click(send())

        expect(files.uploadAttachment).toHaveBeenCalledTimes(1)
        expect(onSend).toHaveBeenLastCalledWith("the globe is black", ["first"])
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
    })

    it("sends a file taken off after it went up with nothing", async () => {
        const {files, landings} = bucket()
        const onSend = vi.fn()
            .mockRejectedValueOnce(new SupportRejectedError())
            .mockResolvedValueOnce(undefined)
        const {user} = setup(onSend, files)

        await writeWith(user, screenshot, recording)
        await user.click(send())
        await act(async () => landings[0].land("first"))
        await act(async () => landings[1].land("second"))
        await user.click(screen.getByRole("button", {name: "Remove screen.png"}))
        await user.click(send())

        expect(onSend).toHaveBeenLastCalledWith("the globe is black", ["second"])
    })

    it("calls the upload off when it is closed, and sends nothing", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {files, landings} = bucket()
        const onSend = vi.fn(async () => undefined)
        const {user} = setup(onSend, files)
        await writeWith(user, screenshot, recording)
        await user.click(send())

        cleanup()
        expect(landings[0].signal?.aborted).toBe(true)

        await act(async () => landings[0].fail())
        expect(files.uploadAttachment).toHaveBeenCalledTimes(1)
        expect(onSend).not.toHaveBeenCalled()
        expect(logged).not.toHaveBeenCalled()
    })

    it("sends nothing when it was closed as the last file landed", async () => {
        const {files, landings} = bucket()
        const onSend = vi.fn(async () => undefined)
        const {user} = setup(onSend, files)
        await writeWith(user, screenshot)
        await user.click(send())

        cleanup()
        await act(async () => landings[0].land("first"))

        expect(onSend).not.toHaveBeenCalled()
    })

    it("still uploads when React mounts it twice, as it does in development", async () => {
        const {files, landings} = bucket()
        const onSend = vi.fn(async () => undefined)
        render(<StrictMode>
            <NoteModal title="Report a bug"
                       label="Your idea"
                       placeholder=""
                       files={files}
                       onSend={onSend}
                       onClose={vi.fn()}/>
        </StrictMode>)
        const user = userEvent.setup()

        await writeWith(user, screenshot)
        await user.click(send())

        expect(landings[0].signal?.aborted).toBe(false)
        await act(async () => landings[0].land("first"))
        expect(onSend).toHaveBeenCalledWith("the globe is black", ["first"])
    })

    it("does not close on a click beside it once a file is picked, or while files go up", async () => {
        const {files, landings} = bucket()
        const {onClose, user} = setup(undefined, files)
        const backdrop = () => document.querySelector(".modal")!

        await addButton()
        await user.upload(input(), [screenshot])
        await user.click(backdrop())
        expect(onClose).not.toHaveBeenCalled()

        await user.type(box(), "the globe is black")
        await user.click(send())
        await user.click(backdrop())
        expect(onClose).not.toHaveBeenCalled()

        await act(async () => landings[0].land("first"))
        await user.click(backdrop())
        expect(onClose).toHaveBeenCalledTimes(1)
    })
})
