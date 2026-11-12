import {ShownNamesBackend} from "./clips.ts"
import {NameplatesBackend} from "./nameplates.ts"

// As if the judge let every username through: a guest, or an account the fakes do not know, is left out.
export class FakeShownNames implements ShownNamesBackend {
    constructor(private readonly nameplates: NameplatesBackend) {
    }

    public async shownNames(accountIds: readonly string[]): Promise<ReadonlyMap<string, string>> {
        const known = await this.nameplates.nameplates([...new Set(accountIds)])
        return new Map(known.flatMap(({accountId, name, guest}) => guest ? [] : [[accountId, name] as const]))
    }
}
