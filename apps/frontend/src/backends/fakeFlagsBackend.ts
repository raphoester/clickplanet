import countriesData from "../../static/countries/countries.json"
import type {ListedFlag} from "../domain/flags.ts"
import {FlagHandlers, FlagLister, FlagStream} from "./flags.ts"

const BUILT_IN: readonly ListedFlag[] = Object.entries(countriesData).map(([code, name]) => ({code, name, closed: false}))

const CODE_LETTERS = "abcdefghijklmnopqrstuvwxyz0123456789"

// `openFlag("Bavaria")` and `closeFlag(code)` on the console, each told as the flags module's stream tells it.
export class FakeFlagsBackend implements FlagLister, FlagStream {
    private readonly opened: ListedFlag[] = []
    private readonly listeners = new Set<FlagHandlers>()

    public async listFlags(): Promise<ListedFlag[]> {
        return [...BUILT_IN, ...this.opened.map((flag) => ({...flag}))]
    }

    public listenForFlags(handlers: FlagHandlers): () => void {
        this.listeners.add(handlers)
        // As the stream's first heartbeat, a moment after it opens.
        const heartbeat = setTimeout(() => handlers.onOpened(), 0)
        return () => {
            clearTimeout(heartbeat)
            this.listeners.delete(handlers)
        }
    }

    public open(name: string): string {
        const code = `r-${Array.from({length: 10}, () => CODE_LETTERS[Math.floor(Math.random() * CODE_LETTERS.length)]).join("")}`
        const flag = {code, name, closed: false}
        this.opened.push(flag)
        this.tell(flag)
        return code
    }

    public close(code: string): boolean {
        const flag = this.opened.find((opened) => opened.code === code)
        if (flag === undefined || flag.closed) return false
        flag.closed = true
        this.tell(flag)
        return true
    }

    private tell(flag: ListedFlag) {
        this.listeners.forEach(({onChanged}) => onChanged({...flag}))
    }
}
