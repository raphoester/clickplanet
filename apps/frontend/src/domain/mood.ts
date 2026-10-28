import {ClickBudget} from "../backends/clickBudget.ts"
import {Battle} from "./battle.ts"

export type Mood = "battle" | "frozen"

// The one place the screen's mood is decided. Frozen wins: planet takes no play, whatever the schedule's turn says.
export function moodOf(battle: Battle | undefined, budget: ClickBudget | undefined): Mood | undefined {
    if (budget?.frozen) return "frozen"
    if (battle) return "battle"
    return undefined
}
