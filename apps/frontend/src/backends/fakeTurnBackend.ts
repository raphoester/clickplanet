import {PLAIN_TURN, Turn, TurnBackend} from "./turn.ts"

const MINUTE_MS = 60_000

export class FakeTurnBackend implements TurnBackend {
    private current: Turn = PLAIN_TURN
    private readonly listeners = new Set<() => void>()

    public async turn(): Promise<Turn> {
        return this.current
    }

    public changes(onChange: () => void): () => void {
        this.listeners.add(onChange)
        return () => {
            this.listeners.delete(onChange)
        }
    }

    public startBattle(minutes = 120, description = "Triple points. A bonus box every minute."): void {
        const now = Date.now()
        this.current = {kind: "battle", since: now, endsAt: now + minutes * MINUTE_MS, description}
        this.tell()
    }

    public endBattle(): void {
        this.current = PLAIN_TURN
        this.tell()
    }

    private tell() {
        for (const listener of this.listeners) listener()
    }
}
