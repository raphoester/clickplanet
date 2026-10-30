import {describe, expect, it} from "vitest"
import {AccountCarrier, carriedFetch} from "./accountCarrier.ts"

function memoryCarrier(held?: string): AccountCarrier & {held?: string} {
    const carrier = {
        held,
        read: () => carrier.held,
        write: (session: string | undefined) => {
            carrier.held = session
        },
    }
    return carrier
}

function answering(headers: Record<string, string>) {
    const sent: RequestInit[] = []
    const fetch: typeof globalThis.fetch = async (_input, init) => {
        sent.push(init ?? {})
        return new Response("{}", {headers})
    }
    return {fetch, sent}
}

describe("carriedFetch", () => {
    it("sends the session it holds, never a cookie", async () => {
        const server = answering({})
        await carriedFetch(memoryCarrier("the-token"), server.fetch)("https://api.example/auth.v1.AuthService/GetMe", {headers: {"Content-Type": "application/proto"}})

        const headers = new Headers(server.sent[0].headers)
        expect(headers.get("X-Account-Session")).toBe("the-token")
        expect(headers.get("Content-Type")).toBe("application/proto")
        expect(server.sent[0].credentials).toBe("omit")
    })

    it("says it holds none before the first one", async () => {
        const server = answering({})
        await carriedFetch(memoryCarrier(), server.fetch)("https://api.example/x", {})

        expect(new Headers(server.sent[0].headers).get("X-Account-Session")).toBe("none")
    })

    it("keeps the session the server answers, and forgets one it clears", async () => {
        const carrier = memoryCarrier("old")

        await carriedFetch(carrier, answering({"X-Account-Session": "new"}).fetch)("https://api.example/x", {})
        expect(carrier.held).toBe("new")

        await carriedFetch(carrier, answering({}).fetch)("https://api.example/x", {})
        expect(carrier.held).toBe("new")

        await carriedFetch(carrier, answering({"X-Account-Session": "none"}).fetch)("https://api.example/x", {})
        expect(carrier.held).toBeUndefined()
    })
})
