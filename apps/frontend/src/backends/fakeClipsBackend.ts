import {Appearance, ClipsBackend} from "./clips.ts"

export class FakeClipsBackend implements ClipsBackend {
    private posted: Appearance[] = []

    constructor(private readonly now: () => number = () => Date.now()) {
    }

    public async appearances(): Promise<Appearance[]> {
        return [...this.posted]
    }

    public giveClip(title = "FRANCE VS TURKEY"): Appearance {
        const appearance: Appearance = {
            postId: `fake-post-${crypto.randomUUID()}`,
            title: title.toUpperCase(),
            scheduledAt: this.now(),
            links: [{platform: "tiktok", url: ""}, {platform: "youtube", url: ""}],
        }
        this.posted = [appearance, ...this.posted]
        return appearance
    }
}
