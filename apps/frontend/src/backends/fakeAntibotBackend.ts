import {WarningAcknowledger, WarningsListener} from "./antibot.ts"

export class FakeAntibotBackend implements WarningsListener, WarningAcknowledger {
    private warned = false
    private readonly listeners = new Set<() => void>()

    public heldIdentity(): string | undefined {
        return "fake-session"
    }

    public warn(): void {
        this.warned = true
        this.listeners.forEach((listener) => listener())
    }

    public listenForWarnings(onWarned: () => void): () => void {
        this.listeners.add(onWarned)
        if (this.warned) onWarned()
        return () => this.listeners.delete(onWarned)
    }

    public async acknowledgeWarning(): Promise<void> {
        this.warned = false
    }
}
