import {GifChoice, GifPage, GifQuery, GifSearch} from "./gifs.ts";
import {GIFS_PER_PAGE} from "./klipyGifs.ts";

const PAGES = 3

const SHAPES = [[220, 220], [220, 124], [220, 290], [220, 165]]

const COLORS = ["#e63946", "#2a9d8f", "#e9c46a", "#457b9d", "#f4a261", "#8d5bd6"]

// "nothing" finds nothing, "fail" fails.
export class FakeGifs implements GifSearch {
    public async find(query: GifQuery, signal?: AbortSignal): Promise<GifPage> {
        await new Promise((resolve) => setTimeout(resolve, 300))
        signal?.throwIfAborted()

        const words = query.words.trim().toLowerCase()
        if (words === "fail") throw new Error("the fake KLIPY fails on purpose")
        if (words === "nothing") return {choices: [], more: false}

        const choices = Array.from({length: GIFS_PER_PAGE}, (_, i) => choiceOf(words || "trending", (query.page - 1) * GIFS_PER_PAGE + i))
        return {choices, more: query.page < PAGES}
    }

    public markShared(choice: GifChoice, query: GifQuery): void {
        console.info("fake KLIPY: shared", choice.slug, "for", JSON.stringify(query.words))
    }
}

function choiceOf(words: string, n: number): GifChoice {
    const [width, height] = SHAPES[n % SHAPES.length]
    const color = COLORS[n % COLORS.length]
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">`
        + `<rect width="100%" height="100%" fill="${color}"/>`
        + `<circle cx="24" cy="${height / 2}" r="16" fill="#fff"><animate attributeName="cx" values="24;${width - 24};24" dur="1.6s" repeatCount="indefinite"/></circle>`
        + `<text x="50%" y="24" text-anchor="middle" font-family="sans-serif" font-size="16" fill="#fff">${escaped(words)} ${n + 1}</text>`
        + `</svg>`
    return {slug: `${words}-${n + 1}`, title: `${words} ${n + 1}`, gif: {url: `data:image/svg+xml,${encodeURIComponent(svg)}`, width, height}}
}

function escaped(text: string): string {
    return text.replace(/[<>&"]/g, (c) => `&#${c.charCodeAt(0)};`)
}
