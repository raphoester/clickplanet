export type Value = {
    code: string,
    name: string
}

/** Whether a value is a result for `search`; an empty search matches everything. */
export function matchesSearch(value: Value, search: string): boolean {
    const term = search.trim().toLowerCase()
    return term === "" || value.name.toLowerCase().includes(term)
}

/**
 * The search results, with the current selection kept among them.
 *
 * This started as a correctness fix for the `<select>` this list used to be: an
 * element whose value matches none of its options does not show nothing
 * selected — the browser selects the *first* option instead, which desynced it
 * from React and made clicking that first option a no-op that fired no `change`
 * event, so the top result of any search could not be picked at all.
 *
 * The list is now our own markup, where every row carries its own handler, so
 * that invariant no longer binds. Keeping the selection visible while the
 * search narrows the list is worth having on its own, and the pinned row is
 * never the one Enter picks — the caller puts its cursor on the first match.
 */
export function visibleOptions(values: Value[], selected: Value, search: string): Value[] {
    const matches = values.filter((v) => matchesSearch(v, search))

    if (matches.some((v) => v.code === selected.code)) return matches
    return [selected, ...matches]
}
