export type Outcome = "taken" | "cleared" | "unchanged"

// Copy of the server's clicks.OutcomeOf: change both together.
export function outcomeOf(owner: string | undefined, ground: string | undefined, flag: string): Outcome {
    if (owner === flag) return "unchanged"
    if (ground !== undefined && owner === ground) return "cleared"
    return "taken"
}

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
