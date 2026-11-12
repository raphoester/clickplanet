import {describe, expect, it} from "vitest"
import {UnaryRequest, UnaryResponse} from "@connectrpc/connect"
import {languageInterceptor} from "./language.ts"

describe("the language interceptor", () => {
    it("sends the page's language with each call", async () => {
        const heard: (string | null)[] = []
        const call = languageInterceptor(Promise.resolve("pt-BR"))(async (req) => {
            heard.push(req.header.get("Accept-Language"))
            return {} as UnaryResponse
        })

        await call({header: new Headers()} as UnaryRequest)

        expect(heard).toEqual(["pt-BR"])
    })
})
