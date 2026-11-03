import {Gif} from "./chat.ts";

export type GifChoice = {
    slug: string
    title: string
    gif: Gif
}

export type GifQuery = {
    // None for what is trending.
    words: string
    page: number
    locale: string
    customer: string
}

export type GifPage = {
    choices: GifChoice[]
    more: boolean
}

export interface GifSearch {
    find(query: GifQuery, signal?: AbortSignal): Promise<GifPage>
    markShared(choice: GifChoice, query: GifQuery): void
}
