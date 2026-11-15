import {NoticeAcknowledger, WarningAcknowledger, WarningsListener} from "./antibot.ts"

type Listener = {onWarned: () => void, onNoticed: () => void}

export class FakeAntibotBackend implements WarningsListener, WarningAcknowledger, NoticeAcknowledger {
    private warned = false
    private noticed = false
    private readonly listeners = new Set<Listener>()

    public heldIdentity(): string | undefined {
        return "fake-session"
    }

    public warn(): void {
        this.warned = true
        this.listeners.forEach((listener) => listener.onWarned())
    }

    public notice(): void {
        this.noticed = true
        this.listeners.forEach((listener) => listener.onNoticed())
    }

    public listenForWarnings(onWarned: () => void, onNoticed: () => void): () => void {
        const listener = {onWarned, onNoticed}
        this.listeners.add(listener)
        if (this.warned) onWarned()
        if (this.noticed) onNoticed()
        return () => this.listeners.delete(listener)
    }

    public async acknowledgeWarning(): Promise<void> {
        this.warned = false
    }

    public async acknowledgeNotice(): Promise<void> {
        this.noticed = false
    }
}
