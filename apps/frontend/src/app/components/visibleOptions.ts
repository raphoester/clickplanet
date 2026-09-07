export type Value = {
    code: string,
    name: string
}

/**
 * The search results, with the current selection kept among them.
 *
 * A `<select>` whose value matches none of its options does not show nothing
 * selected — the browser selects the *first* option instead. That desyncs the
 * element from React, and worse, it means clicking that first option changes
 * nothing and so fires no `change` event: the top result of any search, and the
 * only result of a narrow one, could not be picked at all.
 *
 * Keeping the selection in the list holds the invariant a controlled `<select>`
 * depends on — its value is always one of its options — so every result the
 * search turns up is something the user can actually click.
 */
export function visibleOptions(values: Value[], selected: Value, search: string): Value[] {
    const term = search.trim().toLowerCase()
    const matches = term === ""
        ? values
        : values.filter((v) => v.name.toLowerCase().includes(term))

    if (matches.some((v) => v.code === selected.code)) return matches
    return [selected, ...matches]
}
