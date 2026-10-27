import {PLAIN_TURN, Turn, TurnBackend} from "./turn.ts"

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

    public startBattle(entryId: string, since: number, endsAt: number, description = "Triple points. A bonus box every minute."): void {
        this.current = {kind: "battle", since, endsAt, description, entryId}
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
