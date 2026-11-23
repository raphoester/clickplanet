export type Laying = "lay" | "refused" | "click"

// What a tap does while a mine or a trap is aimed: a tile of another flag, or of none, is clicked first, as with shield
// on; a tile of the flag takes it unless a mine or a trap is on it already, or, for a mine, it is a capital whose take
// would knock the flag out. A trap goes on such a capital: the take springs it first.
export function layingOf(owner: string | undefined, flag: string, armed: boolean, knockOutCapital: boolean): Laying {
    if (owner !== flag) return "click"
    return armed || knockOutCapital ? "refused" : "lay"
}
