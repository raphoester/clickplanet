export type Laying = "lay" | "refused" | "click"

// What a tap does while a mine or a trap is aimed: a tile of another flag, or of none, is clicked first, as with shield
// on; a tile of the flag takes it unless one of its kind is on it already (a mine and a trap share a tile), or, for a
// mine, it is a capital whose take would knock the flag out. A trap goes on such a capital: the take springs it first.
export function layingOf(owner: string | undefined, flag: string, held: boolean, knockOutCapital: boolean): Laying {
    if (owner !== flag) return "click"
    return held || knockOutCapital ? "refused" : "lay"
}
