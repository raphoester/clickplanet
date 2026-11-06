import {useCallback, useEffect, useState} from "react"
import {PromoBackend, TeaserCard} from "../../backends/promo.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {BATTLE_TEASED_STORAGE_KEY, battleTeaserDue, Teased, teasedOf, TEASER_AFTER_CLICKS, teaserKeyOf} from "../../domain/battleTeaser.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"

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

export type Teaser = {battle: BattleToCome, cards: TeaserCard[]}

export type BattleTeaserState = {
    teaser?: Teaser
    dismiss: (battle: BattleToCome) => void
}

// The next battle's teaser, when the time rules say so and promo has cards for it. `guided` is a first visit, which
// plays its clicks before a full screen talks of a battle. A battle with no cards is asked again on the next page.
export function useBattleTeaser(
    battle: BattleToCome | undefined,
    now: number,
    guided: boolean,
    listenForClicks: ListenForClicks,
    promo: PromoBackend,
): BattleTeaserState {
    const [teased, setTeased] = useState(readTeased)
    const [clicks, setClicks] = useState(0)
    const ready = !guided || clicks >= TEASER_AFTER_CLICKS
    const [read, setRead] = useState<{key: string, cards: TeaserCard[]}>()

    useEffect(() => {
        if (ready) return
        return listenForClicks(() => setClicks((count) => count + 1))
    }, [listenForClicks, ready])

    const due = battleTeaserDue(battle, teased, now, ready)
    const key = due ? teaserKeyOf(battle) : undefined
    const readKey = read?.key

    useEffect(() => {
        if (key === undefined || readKey === key) return
        let live = true
        promo.teaser(key).then(
            (cards) => {
                if (live) setRead({key, cards})
            },
            (e: unknown) => {
                console.error("Could not read the battle's teaser", e)
                if (live) setRead({key, cards: []})
            },
        )
        return () => {
            live = false
        }
    }, [key, promo, readKey])

    const dismiss = useCallback((shown: BattleToCome) => {
        const kept = {key: teaserKeyOf(shown), at: Date.now()}
        writeTeased(kept)
        setTeased(kept)
    }, [])

    const cards = read !== undefined && read.key === key ? read.cards : []
    return {teaser: due && cards.length > 0 ? {battle, cards} : undefined, dismiss}
}
