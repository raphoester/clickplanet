// @vitest-environment jsdom
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {NoteAttachment} from "../gen/grpc/support/v1/admin_pb.ts"
import {Attachments} from "./Attachments.tsx"

afterEach(cleanup)

const BUCKET = "https://bucket.example/clickplanet-support"

function file(contentType: string, id: string): NoteAttachment {
    return new NoteAttachment({contentType, sizeBytes: 2048n, url: `${BUCKET}/${id}?X-Amz-Signature=abc`})
}

describe("Attachments", () => {
    it("shows a photo, which opens the file", () => {
        render(<Attachments files={[file("image/png", "a")]}/>)

        const photo = screen.getByRole("img", {name: "Photo 1"})
        expect(photo.getAttribute("src")).toBe(`${BUCKET}/a?X-Amz-Signature=abc`)
        expect(photo.closest("a")?.getAttribute("href")).toBe(`${BUCKET}/a?X-Amz-Signature=abc`)
        expect(screen.getByText("image/png · 2 KB")).toBeTruthy()
    })

    it("plays a video", () => {
        render(<Attachments files={[file("image/png", "a"), file("video/mp4", "b")]}/>)

        const video = screen.getByLabelText("Video 2")
        expect(video.tagName).toBe("VIDEO")
        expect(video.getAttribute("src")).toBe(`${BUCKET}/b?X-Amz-Signature=abc`)
    })

    it("offers the link to a file the browser does not show", () => {
        render(<Attachments files={[file("image/heic", "a")]}/>)

        fireEvent.error(screen.getByRole("img", {name: "Photo 1"}))

        expect(screen.queryByRole("img")).toBeNull()
        expect(screen.getByText(/Photo 1 is not shown here/)).toBeTruthy()
        expect(screen.getByRole("link", {name: "Open"}).getAttribute("href")).toBe(`${BUCKET}/a?X-Amz-Signature=abc`)
    })

    it("shows nothing for a note with no file", () => {
        const {container} = render(<Attachments files={[]}/>)
        expect(container.innerHTML).toBe("")
    })
})
