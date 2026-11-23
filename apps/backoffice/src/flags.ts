import {useSyncExternalStore} from "react"
import {Code, ConnectError} from "@connectrpc/connect"
import type {FlagsAdmin} from "./clients.ts"
import {Flag} from "./gen/grpc/flags/v1/flags_pb.ts"

// A flag as the server lists it: the game's own, then the ones opened since, each named in English, a sold flag with
// its picture.
export type ListedFlag = {code: string, name: string, closed: boolean, picture?: string}

let listed: ReadonlyMap<string, ListedFlag> = new Map()
const listeners = new Set<() => void>()

// Read once, at load: the page has no stream to say the list changed. The Flags page reads it again on each change.
export async function readFlags(admin: FlagsAdmin, signal?: AbortSignal): Promise<void> {
    try {
        const {flags} = await admin.listFlags({}, {signal})
        learnFlags(listedOf(flags))
    } catch (e) {
        if (signal?.aborted) return
        // A server from before the flags module opened none.
        if (e instanceof ConnectError && e.code === Code.Unimplemented) return
        console.error("could not read the list of flags: a flag opened since reads as its code", e)
    }
}

export function listedOf(flags: readonly Flag[]): ListedFlag[] {
    return flags.map(({code, name, closed, sold}) => ({code, name, closed, picture: sold?.picture?.url}))
}

export function learnFlags(flags: readonly ListedFlag[]): void {
    listed = new Map(flags.map(flag => [flag.code, flag]))
    listeners.forEach(listener => listener())
}

export function listedFlag(code: string): ListedFlag | undefined {
    return listed.get(code)
}

// A component that names flags in its render draws again once the list is read.
export function useListedFlags(): ReadonlyMap<string, ListedFlag> {
    return useSyncExternalStore(subscribe, () => listed)
}

function subscribe(listener: () => void): () => void {
    listeners.add(listener)
    return () => listeners.delete(listener)
}
