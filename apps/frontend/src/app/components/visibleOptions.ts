export type Value = {
    code: string,
    name: string
}

export function matchesSearch(value: Value, search: string): boolean {
    const term = search.trim().toLowerCase()
    return term === "" || value.name.toLowerCase().includes(term)
}

export function visibleOptions(values: Value[], selected: Value, search: string): Value[] {
    const matches = values.filter((v) => matchesSearch(v, search))

    if (matches.some((v) => v.code === selected.code)) return matches
    return [selected, ...matches]
}
