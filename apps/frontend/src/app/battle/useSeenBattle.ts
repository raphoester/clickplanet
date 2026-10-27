import {useState} from "react"
import {Battle, battleKeyOf} from "../../domain/battle.ts"

// The last battle the page saw play, kept once it ended.
export function useSeenBattle(battle: Battle | undefined): Battle | undefined {
    const [seen, setSeen] = useState(battle)
    if (battle && (!seen || battleKeyOf(battle) !== battleKeyOf(seen) || battle.description !== seen.description)) setSeen(battle)
    return battle ?? seen
}
