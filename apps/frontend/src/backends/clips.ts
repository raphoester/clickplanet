export type Platform = "tiktok" | "youtube"

export type ClipLink = {
    platform: Platform
    // Empty until the video's address is known.
    url: string
}

export type Appearance = {
    postId: string
    title: string
    scheduledAt: number
    links: ClipLink[]
}

export interface ClipsBackend {
    appearances(): Promise<Appearance[]>
}
