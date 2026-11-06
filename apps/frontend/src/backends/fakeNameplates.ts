import {AccountNameplate, MAX_NAMEPLATES_ASKED, Nameplate, NameplatesBackend} from "./nameplates.ts"

// A fake that knows some accounts' nameplates: undefined for an account it does not know.
export type FakeNameplateSource = {
    nameplateOf(accountId: string): Nameplate | undefined
}

// GetNameplates over the fakes: the first that knows an account answers for it, so the roster, the chat and the boards
// draw one name for one account. As the server, it refuses too many ids, and leaves out an account nobody knows.
export class FakeNameplates implements NameplatesBackend {
    // How many GetNameplates calls the page made: `fakeNameplates.calls` in the console.
    public calls = 0

    constructor(private readonly sources: readonly FakeNameplateSource[]) {
    }

    public async nameplates(accountIds: readonly string[]): Promise<AccountNameplate[]> {
        this.calls++
        if (accountIds.length > MAX_NAMEPLATES_ASKED) throw new Error(`GetNameplates asked ${accountIds.length} accounts, at most ${MAX_NAMEPLATES_ASKED}`)

        return [...new Set(accountIds)].flatMap((accountId) => {
            const nameplate = this.sources.map((source) => source.nameplateOf(accountId)).find((known) => known !== undefined)
            return nameplate ? [{accountId, ...nameplate}] : []
        })
    }
}
