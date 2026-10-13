import {afterEach, describe, expect, it, vi} from "vitest"
import {putFile} from "./putFile.ts"

class FakeRequest {
    static last: FakeRequest

    method = ""
    url = ""
    headers: Record<string, string> = {}
    body?: Blob
    status = 0
    aborted = false
    upload: {onprogress?: (event: {lengthComputable: boolean, loaded: number, total: number}) => void} = {}
    onload?: () => void
    onerror?: () => void
    onabort?: () => void

    constructor() {
        FakeRequest.last = this
    }

    open(method: string, url: string) {
        this.method = method
        this.url = url
    }

    setRequestHeader(name: string, value: string) {
        this.headers[name] = value
    }

    send(body: Blob) {
        this.body = body
    }

    abort() {
        this.aborted = true
        this.onabort?.()
    }

    answer(status: number) {
        this.status = status
        this.onload?.()
    }
}

const screenshot = new Blob(["a screenshot"], {type: "image/png"})

afterEach(() => vi.unstubAllGlobals())

function put(options?: Parameters<typeof putFile>[2]) {
    vi.stubGlobal("XMLHttpRequest", FakeRequest)
    const sent = putFile("https://bucket.example/a/0?X-Amz-Signature=abc", screenshot, options)
    return {sent, request: FakeRequest.last}
}

describe("putFile", () => {
    it("puts the file at the link with the type the link was signed for", async () => {
        const {sent, request} = put()

        request.answer(200)
        await sent

        expect(request.method).toBe("PUT")
        expect(request.url).toBe("https://bucket.example/a/0?X-Amz-Signature=abc")
        expect(request.headers).toEqual({"Content-Type": "image/png"})
        expect(request.body).toBe(screenshot)
    })

    it("says how far the file is", async () => {
        const onProgress = vi.fn()
        const {sent, request} = put({onProgress})

        request.upload.onprogress?.({lengthComputable: true, loaded: 3, total: 12})
        request.upload.onprogress?.({lengthComputable: false, loaded: 6, total: 0})
        request.answer(200)
        await sent

        expect(onProgress.mock.calls).toEqual([[0.25]])
    })

    it("fails when the bucket refuses the file", async () => {
        const {sent, request} = put()

        request.answer(403)

        await expect(sent).rejects.toThrow("403")
    })

    it("fails when the file could not be sent", async () => {
        const {sent, request} = put()

        request.onerror?.()

        await expect(sent).rejects.toThrow("could not be sent")
    })

    it("stops when it is called off", async () => {
        const calledOff = new AbortController()
        const {sent, request} = put({signal: calledOff.signal})

        calledOff.abort()

        await expect(sent).rejects.toThrow("called off")
        expect(request.aborted).toBe(true)
    })

    it("sends nothing once it was called off", async () => {
        const calledOff = new AbortController()
        calledOff.abort()
        vi.stubGlobal("XMLHttpRequest", vi.fn(() => {
            throw new Error("no request should be made")
        }))

        await expect(putFile("https://bucket.example/a/0", screenshot, {signal: calledOff.signal})).rejects.toThrow("called off")
    })
})
