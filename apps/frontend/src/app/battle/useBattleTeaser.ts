import {useCallback, useEffect, useState} from "react"
import {TeaserCard} from "../../backends/promo.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {BATTLE_TEASED_STORAGE_KEY, battleTeaserDue, Teased, teasedOf, TEASER_AFTER_CLICKS, teaserKeyOf} from "../../domain/battleTeaser.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useBattleCards} from "./teaserCardsStore.ts"

function readTeased(): Teased | null {
    try {
        return teasedOf(window.localStorage.getItem(BATTLE_TEASED_STORAGE_KEY))
    } catch {
        return null
    }
}

function writeTeased(teased: Teased) {
    try {
        window.localStorage.setItem(BATTLE_TEASED_STORAGE_KEY, JSON.stringify(teased))
    } catch (e) {
        console.error("Could not remember the battle teased", e)
    }
}

export type Teaser = {battle: BattleToCome, cards: readonly TeaserCard[]}

export type BattleTeaserState = {
    teaser?: Teaser
    dismiss: (battle: BattleToCome) => void
}

// The next battle's teaser, when the time rules say so and promo has cards for it (`useTeaserCards`). `guided` is a
// first visit, which plays its clicks before a full screen talks of a battle. A battle with no cards is asked again on
// the next page.
export function useBattleTeaser(
    battle: BattleToCome | undefined,
    now: number,
    guided: boolean,
    listenForClicks: ListenForClicks,
): BattleTeaserState {
    const [teased, setTeased] = useState(readTeased)
    const [clicks, setClicks] = useState(0)
    const ready = !guided || clicks >= TEASER_AFTER_CLICKS

    useEffect(() => {
        if (ready) return
        return listenForClicks(() => setClicks((count) => count + 1))
    }, [listenForClicks, ready])

    const due = battleTeaserDue(battle, teased, now, ready)
    const cards = useBattleCards(due ? teaserKeyOf(battle) : undefined)

    // Told is told, whether the teaser came on its own or the player asked for it. A battle that started is never
    // teased again, and its mark would only hide the next one's: it is not kept.
    const dismiss = useCallback((shown: BattleToCome) => {
        if (shown.startsAt <= Date.now()) return
        const kept = {key: teaserKeyOf(shown), at: Date.now()}
        writeTeased(kept)
        setTeased(kept)
    }, [])

    return {teaser: due && cards !== undefined && cards.length > 0 ? {battle, cards} : undefined, dismiss}
}
