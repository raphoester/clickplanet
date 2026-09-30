/**
 * Native land takes two clicks — the server's `clicks.OutcomeOf`, copied so a
 * click is painted as the server will write it.
 *
 * On a country's own ground, a tile wearing that country's flag is cleared to
 * nobody by a click for any other flag, not taken; the next click on the empty
 * tile takes it. Its natives win it back in one. Every click costs one token
 * whatever it does. `homeSoil.test.ts` holds the same cases as the server's
 * `home_soil_test.go`: change one, change both.
 *
 * - `taken`: the tile wears the flag clicked.
 * - `cleared`: the tile belongs to nobody.
 * - `unchanged`: the tile already wore the flag.
 */
export type Outcome = "taken" | "cleared" | "unchanged"

/**
 * What a click for `flag` does to a tile `owner` holds, on the ground of
 * `ground`. Nobody, on either side, is `undefined`: a tile in no country is
 * never anybody's home.
 */
export function outcomeOf(owner: string | undefined, ground: string | undefined, flag: string): Outcome {
    if (owner === flag) return "unchanged"
    if (ground !== undefined && owner === ground) return "cleared"
    return "taken"
}

/** Who holds the tile once a click for `flag` had `outcome` on a tile `owner` held. */
export function ownerAfter(outcome: Outcome, owner: string | undefined, flag: string): string | undefined {
    switch (outcome) {
        case "taken":
            return flag
        case "cleared":
            return undefined
        case "unchanged":
            return owner
    }
}
