// A player, as the planet's tools take it: an account, or a scope (an address, read as its /32 or /64) for the
// takes made with no account and for every account behind it. Never both: the server refuses that.
export type Caller = {accountId: string} | {scope: string}

const ACCOUNT_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

// An account id is a UUID, and anything else the operator typed is an address. The server checks either.
export function callerOf(typed: string): Caller | undefined {
    const text = typed.trim()
    if (text === "") return undefined
    return ACCOUNT_ID.test(text) ? {accountId: text.toLowerCase()} : {scope: text}
}

export function isAccount(caller: Caller): caller is {accountId: string} {
    return "accountId" in caller
}

export function idOf(caller: Caller): string {
    return isAccount(caller) ? caller.accountId : caller.scope
}

// How a question names the player: by its name when the page has one, never by an id nobody reads.
export function whoOf(caller: Caller, name = ""): string {
    if (name !== "") return name
    return isAccount(caller) ? "this account" : "this address"
}
