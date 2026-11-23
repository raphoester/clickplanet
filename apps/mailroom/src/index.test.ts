import {afterEach, describe, expect, it, vi} from "vitest"
import {deliver, Env, Mail} from "./index.ts"

const env: Env = {MAILROOM_URL: "https://api.example/mailroom/replies", MAILROOM_SECRET: "the-secret"}

function mail(raw: string) {
    const rejected: string[] = []
    const sent: Mail = {
        to: "support+1@clickplanet.lol",
        raw: new Blob([raw]).stream(),
        setReject: (reason: string) => { rejected.push(reason) },
    }
    return {sent, rejected}
}

function answering(status: number) {
    const fetched = vi.fn(async () => new Response(null, {status}))
    vi.stubGlobal("fetch", fetched)
    return fetched
}

afterEach(() => vi.unstubAllGlobals())

describe("deliver", () => {
    it("posts the raw mail and its envelope recipient with the secret", async () => {
        const fetched = answering(202)
        const {sent, rejected} = mail("Message-ID: <1@x>\r\n\r\nHi")

        await deliver(sent, env)

        expect(fetched).toHaveBeenCalledOnce()
        const [url, init] = fetched.mock.calls[0] as unknown as [string, RequestInit]
        expect(url).toBe("https://api.example/mailroom/replies")
        expect(init.method).toBe("POST")
        expect(init.headers).toEqual({
            "Authorization": "Bearer the-secret",
            "Content-Type": "message/rfc822",
            "X-Envelope-To": "support+1@clickplanet.lol",
        })
        expect(new TextDecoder().decode(init.body as ArrayBuffer)).toBe("Message-ID: <1@x>\r\n\r\nHi")
        expect(rejected).toEqual([])
    })

    it("fails so the mail comes again when the backend did not keep it", async () => {
        for (const status of [401, 500, 502]) {
            answering(status)

            await expect(deliver(mail("Hi").sent, env)).rejects.toThrow(String(status))
        }
    })

    it("fails when the backend cannot be reached", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => { throw new TypeError("network") }))

        await expect(deliver(mail("Hi").sent, env)).rejects.toThrow("network")
    })

    it("refuses for good a mail the backend finds too large", async () => {
        answering(413)
        const {sent, rejected} = mail("Hi")

        await deliver(sent, env)

        expect(rejected).toEqual(["This mail is too large."])
    })
})
