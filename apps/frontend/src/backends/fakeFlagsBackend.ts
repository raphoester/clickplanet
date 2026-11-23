import countriesData from "../../static/countries/countries.json"
import type {ListedFlag} from "../domain/flags.ts"
import {FlagChanges, FlagLister} from "./flags.ts"

const BUILT_IN: readonly ListedFlag[] = Object.entries(countriesData).map(([code, name]) => ({code, name, closed: false}))

const CODE_LETTERS = "abcdefghijklmnopqrstuvwxyz0123456789"

// `openFlag("Bavaria")` and `closeFlag(code)` on the console, each told as planet's stream tells it.
export class FakeFlagsBackend implements FlagLister, FlagChanges {
    private readonly opened: ListedFlag[] = []
    private readonly listeners = new Set<() => void>()

    public async listFlags(): Promise<ListedFlag[]> {
        return [...BUILT_IN, ...this.opened.map((flag) => ({...flag}))]
    }

    public listenForFlagChanges(onChanged: () => void): () => void {
        const listener = () => onChanged()
        this.listeners.add(listener)
        return () => this.listeners.delete(listener)
    }

    public open(name: string): string {
        const code = `r-${Array.from({length: 10}, () => CODE_LETTERS[Math.floor(Math.random() * CODE_LETTERS.length)]).join("")}`
        this.opened.push({code, name, closed: false})
        this.tell()
        return code
    }

    public close(code: string): boolean {
        const flag = this.opened.find((opened) => opened.code === code)
        if (flag === undefined || flag.closed) return false
        flag.closed = true
        this.tell()
        return true
    }

    private tell() {
        this.listeners.forEach((listener) => listener())
    }
}
