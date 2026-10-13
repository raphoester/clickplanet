// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {
    AttachmentRules,
    AttachmentUpload,
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

type Send = (text: string, attachments: File[]) => Promise<AttachmentUpload[]>

function setup(onSend: Send = vi.fn(async () => []), files?: NoteFiles) {
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
        const onSend = vi.fn(() => new Promise<AttachmentUpload[]>((resolve) => {
            land = () => resolve([])
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
        [new SupportRateLimitedError(), "You sent too many. Try again in a few minutes."],
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

type Landing = {land: () => void, fail: () => void, progress: (fraction: number) => void, signal?: AbortSignal}

function bucket(read: AttachmentRules | Error = rules) {
    const landings: Landing[] = []
    const files = {
        attachmentRules: vi.fn(async () => {
            if (read instanceof Error) throw read
            return read
        }),
        uploadAttachment: vi.fn((_: AttachmentUpload, __: Blob, options?: UploadOptions) => new Promise<void>((resolve, reject) => {
            landings.push({
                land: resolve,
                fail: () => reject(new Error("the upload was answered 403")),
                progress: (fraction) => options?.onProgress?.(fraction),
                signal: options?.signal,
            })
        })),
    }
    return {files, landings}
}

const sending = (...urls: string[]): Send => vi.fn(async () => urls.map((url) => ({url})))

const addButton = () => screen.findByRole("button", {name: "Add photos or videos"}) as Promise<HTMLButtonElement>
const input = () => document.querySelector("input[type=file]") as HTMLInputElement
const attached = () => within(screen.getByRole("list", {name: "Attached files"})).getAllByRole("listitem")
const uploads = () => within(screen.getByRole("list", {name: "Files"})).getAllByRole("listitem")
const status = () => screen.getByRole("status").textContent

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

    it("sends the note with its files, then uploads each one where the server said, in order", async () => {
        const {files, landings} = bucket()
        const onSend = sending("https://bucket.example/a/0", "https://bucket.example/a/1")
        const {user} = setup(onSend, files)

        await writeWith(user, screenshot, recording)
        await user.click(send())

        expect(onSend).toHaveBeenCalledWith("the globe is black", [screenshot, recording])
        expect(status()).toBe("Sent. Uploading your files…")
        expect(files.uploadAttachment).toHaveBeenCalledTimes(1)
        expect(files.uploadAttachment.mock.calls[0].slice(0, 2)).toEqual([{url: "https://bucket.example/a/0"}, screenshot])
        expect((screen.getByRole("button", {name: "Uploading…"}) as HTMLButtonElement).disabled).toBe(true)

        await act(async () => landings[0].progress(0.5))
        expect(screen.getByRole("progressbar", {name: "Uploading screen.png"}).getAttribute("aria-valuenow")).toBe("50")

        await act(async () => landings[0].land())
        expect(files.uploadAttachment.mock.calls[1].slice(0, 2)).toEqual([{url: "https://bucket.example/a/1"}, recording])
        expect(within(uploads()[0]).getByRole("img", {name: "Uploaded"})).not.toBeNull()

        await act(async () => landings[1].land())
        expect(status()).toBe("Sent. Thank you!")
        expect(screen.queryByRole("progressbar")).toBeNull()
        expect(screen.getAllByRole("button", {name: "Close"}).at(-1)).not.toBeUndefined()
    })

    it("uploads nothing when the note itself was refused, and keeps the files", async () => {
        const {files} = bucket()
        const {user} = setup(vi.fn(async () => {
            throw new SupportRateLimitedError()
        }), files)

        await writeWith(user, screenshot)
        await user.click(send())

        expect(screen.getByRole("alert").textContent).toBe("You sent too many. Try again in a few minutes.")
        expect(files.uploadAttachment).not.toHaveBeenCalled()
        expect(attached()).toHaveLength(1)
    })

    it("says which file did not upload, and tries that one again alone", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {files, landings} = bucket()
        const {user} = setup(sending("https://bucket.example/a/0", "https://bucket.example/a/1"), files)

        await writeWith(user, screenshot, recording)
        await user.click(send())
        await act(async () => landings[0].land())
        await act(async () => landings[1].fail())

        expect(status()).toBe("Sent, but 1 file did not upload.")
        expect(uploads()[1].textContent).toBe("clip.mp4Not uploaded")
        expect(logged).toHaveBeenCalledTimes(1)

        await user.click(screen.getByRole("button", {name: "Try again"}))

        expect(files.uploadAttachment).toHaveBeenCalledTimes(3)
        expect(files.uploadAttachment.mock.calls[2].slice(0, 2)).toEqual([{url: "https://bucket.example/a/1"}, recording])

        await act(async () => landings[2].land())
        expect(status()).toBe("Sent. Thank you!")
        expect(screen.queryByRole("button", {name: "Try again"})).toBeNull()
    })

    it("counts every file that did not upload", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const {files, landings} = bucket()
        const {user} = setup(sending("https://bucket.example/a/0", "https://bucket.example/a/1"), files)

        await writeWith(user, screenshot, recording)
        await user.click(send())
        await act(async () => landings[0].fail())
        await act(async () => landings[1].fail())

        expect(status()).toBe("Sent, but 2 files did not upload.")
    })

    it("counts as not uploaded a file the server gave no place for", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const {files} = bucket()
        const {user} = setup(sending(), files)

        await writeWith(user, screenshot)
        await user.click(send())

        expect(status()).toBe("Sent, but 1 file did not upload.")
        expect(files.uploadAttachment).not.toHaveBeenCalled()
    })

    it("calls the uploads off when it is closed, and says nothing more", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {files, landings} = bucket()
        const {user} = setup(sending("https://bucket.example/a/0", "https://bucket.example/a/1"), files)
        await writeWith(user, screenshot, recording)
        await user.click(send())

        cleanup()
        expect(landings[0].signal?.aborted).toBe(true)

        await act(async () => landings[0].fail())
        expect(files.uploadAttachment).toHaveBeenCalledTimes(1)
        expect(logged).not.toHaveBeenCalled()
    })

    it("uploads nothing when it was closed while the note was on its way", async () => {
        const {files} = bucket()
        let land = () => {}
        const onSend = vi.fn(() => new Promise<AttachmentUpload[]>((resolve) => {
            land = () => resolve([{url: "https://bucket.example/a/0"}])
        }))
        const {user} = setup(onSend, files)
        await writeWith(user, screenshot)
        await user.click(send())

        cleanup()
        await act(async () => land())

        expect(files.uploadAttachment).not.toHaveBeenCalled()
    })

    it("still uploads when React mounts it twice, as it does in development", async () => {
        const {files, landings} = bucket()
        render(<StrictMode>
            <NoteModal title="Report a bug"
                       label="Your idea"
                       placeholder=""
                       files={files}
                       onSend={sending("https://bucket.example/a/0")}
                       onClose={vi.fn()}/>
        </StrictMode>)
        const user = userEvent.setup()

        await writeWith(user, screenshot)
        await user.click(send())

        expect(landings[0].signal?.aborted).toBe(false)
        await act(async () => landings[0].land())
        expect(status()).toBe("Sent. Thank you!")
    })

    it("does not close on a click beside it once a file is picked, or while files go up", async () => {
        const {files, landings} = bucket()
        const {onClose, user} = setup(sending("https://bucket.example/a/0"), files)
        const backdrop = () => document.querySelector(".modal")!

        await addButton()
        await user.upload(input(), [screenshot])
        await user.click(backdrop())
        expect(onClose).not.toHaveBeenCalled()

        await user.type(box(), "the globe is black")
        await user.click(send())
        await user.click(backdrop())
        expect(onClose).not.toHaveBeenCalled()

        await act(async () => landings[0].land())
        await user.click(backdrop())
        expect(onClose).toHaveBeenCalledTimes(1)
    })
})
