import {Appearance, ClipsBackend, ClipsNameBackend} from "./clips.ts"

export class FakeClipsBackend implements ClipsBackend, ClipsNameBackend {
    private posted: Appearance[] = []
    private shown = true

    constructor(private readonly now: () => number = () => Date.now()) {
    }

    public async appearances(): Promise<Appearance[]> {
        return [...this.posted]
    }

    public async nameShown(): Promise<boolean> {
        return this.shown
    }

    public async setNameShown(shown: boolean): Promise<boolean> {
        this.shown = shown
        return shown
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
