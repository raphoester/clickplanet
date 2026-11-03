import {Gif} from "./chat.ts";
import {GifChoice, GifPage, GifQuery, GifSearch} from "./gifs.ts";

const KLIPY_API = "https://api.klipy.com/api/v1"

export const GIFS_PER_PAGE = 24

// Tenor's levels, which KLIPY keeps: medium lets G and PG through.
const CONTENT_FILTER = "medium"

type Fetch = (input: string, init?: RequestInit) => Promise<Response>

// Called from the browser: KLIPY's terms forbid a server between the player and it.
export class KlipyGifs implements GifSearch {
    constructor(
        private readonly key: string,
        private readonly fetcher: Fetch = (input, init) => fetch(input, init),
    ) {
    }

    public async find(query: GifQuery, signal?: AbortSignal): Promise<GifPage> {
        const words = query.words.trim()
        const params = new URLSearchParams({
            page: `${query.page}`,
            per_page: `${GIFS_PER_PAGE}`,
            customer_id: query.customer,
            locale: query.locale,
            content_filter: CONTENT_FILTER,
        })
        if (words !== "") params.set("q", words)

        const res = await this.fetcher(`${this.base()}/${words === "" ? "trending" : "search"}?${params}`, {signal})
        if (!res.ok) throw new Error(`KLIPY answered ${res.status}`)
        return pageOf(await res.json())
    }

    public markShared(choice: GifChoice, query: GifQuery): void {
        this.fetcher(`${this.base()}/share/${encodeURIComponent(choice.slug)}`, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify({customer_id: query.customer, q: query.words.trim()}),
        }).catch((e) => console.error("KLIPY was not told of a GIF sent", e))
    }

    private base(): string {
        return `${KLIPY_API}/${encodeURIComponent(this.key)}/gifs`
    }
}

type KlipyPage = {
    data?: {
        data?: unknown
        has_next?: unknown
    }
}

export function pageOf(body: unknown): GifPage {
    const page = (body as KlipyPage | null)?.data
    const items = Array.isArray(page?.data) ? page.data : []
    return {choices: items.flatMap(choiceOf), more: page?.has_next === true}
}

// The small size, the one a chat line shows: webp, else the GIF.
function choiceOf(item: unknown): GifChoice[] {
    if (typeof item !== "object" || item === null) return []
    const {type, slug, title, file} = item as {type?: unknown, slug?: unknown, title?: unknown, file?: {sm?: {webp?: unknown, gif?: unknown}}}
    if (type === "ad" || typeof slug !== "string") return []

    const gif = mediaOf(file?.sm?.webp) ?? mediaOf(file?.sm?.gif)
    return gif ? [{slug, title: typeof title === "string" ? title : "", gif}] : []
}

function mediaOf(media: unknown): Gif | undefined {
    if (typeof media !== "object" || media === null) return undefined
    const {url, width, height} = media as {url?: unknown, width?: unknown, height?: unknown}
    if (typeof url !== "string" || typeof width !== "number" || typeof height !== "number") return undefined
    if (url === "" || width <= 0 || height <= 0) return undefined
    return {url, width, height}
}
