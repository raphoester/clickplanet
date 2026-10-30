// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {Put} from "./actions.ts"
import {VideosPage} from "./VideosPage.tsx"

afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
})

const fileOf = (name: string, type = "video/mp4") => new File(["0123456789"], name, {type})

function putting() {
    return vi.fn<Put>((_url, file, _type, onSent) => {
        onSent(file.size)
        return Promise.resolve()
    })
}

function pick(...files: File[]) {
    fireEvent.change(screen.getByLabelText("Pick videos"), {target: {files}})
}

describe("VideosPage", () => {
    it("lists each video with its player, caption and download link", async () => {
        const admin = new FakeAdmin()
        render(<VideosPage clips={admin} put={putting()}/>)

        const card = (await screen.findByRole("heading", {name: "season-1-france"})).closest("li") as HTMLElement
        expect(within(card).getByText(/^12\.0 MB/)).toBeTruthy()
        expect(within(card).getByText("France held the map to the end. #clickplanet")).toBeTruthy()
        expect(within(card).getByLabelText("season-1-france").getAttribute("src")).toBe("https://bucket.test/video-1/view")
        expect(within(card).getByRole("link", {name: "Download"}).getAttribute("href"))
            .toBe("https://bucket.test/video-1/download")
        expect(screen.getByText("1 video")).toBeTruthy()
    })

    it("uploads a picked file under its name, then lists it", async () => {
        const admin = new FakeAdmin()
        const put = putting()
        render(<VideosPage clips={admin} put={put}/>)
        await screen.findByText("1 video")

        pick(fileOf("season-1-algeria.mp4"))
        const upload = screen.getByRole("region", {name: "Upload"})
        expect((within(upload).getByLabelText("Name") as HTMLInputElement).value).toBe("season-1-algeria")
        fireEvent.change(within(upload).getByLabelText("Caption"), {target: {value: "Algeria came back. #clickplanet "}})
        fireEvent.click(within(upload).getByRole("button", {name: "Upload"}))

        expect(await within(upload).findByText("Uploaded as season-1-algeria.")).toBeTruthy()
        expect(admin.askedFor("createVideoUpload")).toEqual([{contentType: "video/mp4", sizeBytes: 10n}])
        expect(put).toHaveBeenCalledWith("https://bucket.test/upload-1", expect.any(File), "video/mp4", expect.any(Function))
        expect(admin.askedFor("keepVideo")).toEqual([
            {videoId: "upload-1", name: "season-1-algeria", caption: "Algeria came back. #clickplanet"},
        ])
        expect(await screen.findByRole("heading", {name: "season-1-algeria"})).toBeTruthy()
        expect(screen.getByText("2 videos")).toBeTruthy()
    })

    it("says what the server refused, and sends no file", async () => {
        const admin = new FakeAdmin()
        admin.refuse("createVideoUpload", Code.InvalidArgument)
        const put = putting()
        render(<VideosPage clips={admin} put={put}/>)

        pick(fileOf("clip.mkv", "video/x-matroska"))
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))

        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()
        expect(put).not.toHaveBeenCalled()
    })

    it("says videos are off when the server has them off", async () => {
        const admin = new FakeAdmin()
        admin.refuse("createVideoUpload", Code.FailedPrecondition)
        render(<VideosPage clips={admin} put={putting()}/>)

        pick(fileOf("clip.mp4"))
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))

        expect(await screen.findByText("Videos are off on the server.")).toBeTruthy()
    })

    it("names a sent file again without sending it again", async () => {
        const admin = new FakeAdmin()
        admin.refuse("keepVideo", Code.InvalidArgument)
        const put = putting()
        render(<VideosPage clips={admin} put={put}/>)

        pick(fileOf("clip.mp4"))
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))
        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()

        admin.refusals.delete("keepVideo")
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))

        expect(await screen.findByText("Uploaded as clip.")).toBeTruthy()
        expect(put).toHaveBeenCalledTimes(1)
        expect(admin.askedFor("createVideoUpload")).toHaveLength(1)
    })

    it("sends the file again when the bucket does not hold it", async () => {
        const admin = new FakeAdmin()
        admin.refuse("keepVideo", Code.FailedPrecondition)
        const put = putting()
        render(<VideosPage clips={admin} put={put}/>)

        pick(fileOf("clip.mp4"))
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))
        expect(await screen.findByText("The file is not in the bucket. Upload it again.")).toBeTruthy()

        admin.refusals.delete("keepVideo")
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))

        expect(await screen.findByText("Uploaded as clip.")).toBeTruthy()
        expect(put).toHaveBeenCalledTimes(2)
    })

    it("says the bucket's answer when the upload fails", async () => {
        const admin = new FakeAdmin()
        const put = vi.fn<Put>(() => Promise.reject(new Error("the bucket answered 403: Request has expired")))
        render(<VideosPage clips={admin} put={put}/>)

        pick(fileOf("clip.mp4"))
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))

        expect(await screen.findByText("The upload failed: the bucket answered 403: Request has expired.")).toBeTruthy()
        expect(admin.askedFor("keepVideo")).toEqual([])
    })

    it("uploads nothing without a name", () => {
        render(<VideosPage clips={new FakeAdmin()} put={putting()}/>)

        pick(fileOf("clip.mp4"))
        fireEvent.change(screen.getByLabelText("Name"), {target: {value: "  "}})

        expect((screen.getByRole("button", {name: "Upload"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("removes a video once asked, and says so", async () => {
        const admin = new FakeAdmin()
        render(<VideosPage clips={admin} put={putting()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Remove"}))
        expect(screen.getByText("Remove season-1-france?")).toBeTruthy()
        expect(admin.askedFor("removeVideo")).toEqual([])
        fireEvent.click(screen.getByRole("button", {name: "Yes, remove"}))

        expect(await screen.findByText("Removed season-1-france.")).toBeTruthy()
        expect(admin.askedFor("removeVideo")).toEqual([{videoId: "video-1"}])
        expect(screen.queryByRole("heading", {name: "season-1-france"})).toBeNull()
    })

    it("copies the caption", async () => {
        const writeText = vi.fn(() => Promise.resolve())
        vi.stubGlobal("navigator", {clipboard: {writeText}})
        render(<VideosPage clips={new FakeAdmin()} put={putting()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Copy caption"}))

        expect(await screen.findByRole("button", {name: "Copied"})).toBeTruthy()
        expect(writeText).toHaveBeenCalledWith("France held the map to the end. #clickplanet")
    })

    it("selects the caption when the browser refuses to copy it", async () => {
        vi.stubGlobal("navigator", {clipboard: {writeText: () => Promise.reject(new Error("denied"))}})
        document.execCommand = () => false
        render(<VideosPage clips={new FakeAdmin()} put={putting()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Copy caption"}))

        expect(await screen.findByText(/^The browser refused to copy/)).toBeTruthy()
        expect(window.getSelection()?.toString()).toBe("France held the map to the end. #clickplanet")
    })

    it("shows nothing but that when the server has videos off", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listVideos", Code.FailedPrecondition)
        render(<VideosPage clips={admin} put={putting()}/>)

        expect(await screen.findByText("Videos are off on the server.")).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Upload"})).toBeNull()
    })
})
