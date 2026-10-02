export class OwnClicks {
    private readonly clicks = new Map<number, {country: string, at: number}>()

    constructor(
        private readonly windowSeconds: number,
        private readonly limit = 256,
    ) {
    }

    record(tile: number, country: string, seconds: number) {
        this.clicks.set(tile, {country, at: seconds})
        if (this.clicks.size <= this.limit) return
        for (const [id, click] of this.clicks) {
            if (seconds - click.at > this.windowSeconds) this.clicks.delete(id)
        }
    }

    has(tile: number, country: string, seconds: number): boolean {
        const click = this.clicks.get(tile)
        return click !== undefined && click.country === country && seconds - click.at <= this.windowSeconds
    }
}
