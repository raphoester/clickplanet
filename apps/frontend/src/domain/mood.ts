import {ClickBudget} from "../backends/clickBudget.ts"
import {Turn} from "../backends/turn.ts"
import {battleOf} from "./battle.ts"

export type Mood = "battle" | "frozen" | "halloween"

// The one place the screen's mood is decided. Frozen wins: planet takes no play, whatever the schedule's turn says.
export function moodOf(turn: Turn | undefined, budget: ClickBudget | undefined): Mood | undefined {
    if (budget?.frozen) return "frozen"
    if (battleOf(turn)) return "battle"
    if (turn?.kind === "halloween") return "halloween"
    return undefined
}
