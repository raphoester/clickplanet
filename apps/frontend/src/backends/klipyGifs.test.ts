import {describe, expect, it, vi} from "vitest"
import {GIFS_PER_PAGE, KlipyGifs, pageOf} from "./klipyGifs.ts"

const media = (name: string, width = 220, height = 124) => ({url: `https://static.klipy.com/ii/a/${name}`, width, height, size: 1000})

const item = (slug: string, sm: object) => ({id: 1, slug, title: `title ${slug}`, type: "gif", file: {sm, xs: {webp: media("xs.webp", 90, 51)}}})

const query = {words: "", page: 1, locale: "fr", customer: "browser-1"}

describe("pageOf", () => {
    it("takes the small webp of each GIF, with its size", () => {
        const page = pageOf({result: true, data: {data: [item("hello-1", {webp: media("sm.webp"), gif: media("sm.gif")})], has_next: true}})

        expect(page).toEqual({
            choices: [{slug: "hello-1", title: "title hello-1", gif: {url: media("sm.webp").url, width: 220, height: 124}}],
            more: true,
        })
    })

    it("falls back to the small GIF when there is no webp", () => {
        const page = pageOf({data: {data: [item("hello-1", {gif: media("sm.gif")})]}})

        expect(page.choices[0].gif.url).toBe(media("sm.gif").url)
        expect(page.more).toBe(false)
    })

    it("leaves out an ad, and a GIF with no small file or no size", () => {
        const page = pageOf({data: {data: [
            {...item("ad-1", {webp: media("sm.webp")}), type: "ad"},
            item("empty", {}),
            item("flat", {webp: media("sm.webp", 220, 0)}),
            item("kept", {webp: media("sm.webp")}),
            null,
        ]}})

        expect(page.choices.map(choice => choice.slug)).toEqual(["kept"])
    })

    it("reads a body it does not understand as an empty page", () => {
        expect(pageOf(undefined)).toEqual({choices: [], more: false})
        expect(pageOf({data: "nope"})).toEqual({choices: [], more: false})
    })
})

describe("KlipyGifs", () => {
    const answering = (body: unknown, status = 200) =>
        vi.fn<(input: string, init?: RequestInit) => Promise<Response>>(async () => new Response(JSON.stringify(body), {status}))

    it("asks for what is trending when there are no words", async () => {
        const fetcher = answering({data: {data: []}})

        await new KlipyGifs("key-1", fetcher).find(query)

        const url = new URL(fetcher.mock.calls[0][0])
        expect(url.origin + url.pathname).toBe("https://api.klipy.com/api/v1/key-1/gifs/trending")
        expect(Object.fromEntries(url.searchParams)).toEqual({
            page: "1", per_page: `${GIFS_PER_PAGE}`, customer_id: "browser-1", locale: "fr", content_filter: "medium",
        })
    })

    it("searches the words, trimmed", async () => {
        const fetcher = answering({data: {data: []}})

        await new KlipyGifs("key-1", fetcher).find({...query, words: "  happy dance ", page: 2})

        const url = new URL(fetcher.mock.calls[0][0])
        expect(url.pathname).toBe("/api/v1/key-1/gifs/search")
        expect(url.searchParams.get("q")).toBe("happy dance")
        expect(url.searchParams.get("page")).toBe("2")
    })

    it("fails on an answer that is not a success", async () => {
        await expect(new KlipyGifs("key-1", answering({}, 429)).find(query)).rejects.toThrow("429")
    })

    it("tells KLIPY which GIF was sent, and for which words", async () => {
        const fetcher = answering({result: true})
        const choice = {slug: "hello-1", title: "Hello", gif: {url: media("sm.webp").url, width: 220, height: 124}}

        new KlipyGifs("key-1", fetcher).markShared(choice, {...query, words: " hello "})

        expect(fetcher).toHaveBeenCalledWith("https://api.klipy.com/api/v1/key-1/gifs/share/hello-1", {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify({customer_id: "browser-1", q: "hello"}),
        })
    })
})
