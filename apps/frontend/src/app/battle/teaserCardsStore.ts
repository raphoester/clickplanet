import {useEffect} from "react"
import {create} from "zustand"
import {PromoBackend, TeaserCard} from "../../backends/promo.ts"

type TeaserCardsStore = {
    // By the battle's key (`teaserKeyOf`), the cards promo answered: none for a battle whose set has none, one the
    // schedule no longer holds, or a read that failed. A key never read is absent.
    cards: Readonly<Record<string, readonly TeaserCard[]>>
    // Reads a battle's cards, once a page: cards the operator writes later show on the next page.
    ask: (key: string) => void
    // Where the cards are read from. None stops reading and forgets every card.
    serve: (backend: PromoBackend | undefined) => void
}

export const useTeaserCards = create<TeaserCardsStore>()((set) => {
    let backend: PromoBackend | undefined
    // Every key asked, read once a backend serves them: a component asks before the page's feed serves it.
    const asked = new Set<string>()
    const reading = new Set<string>()

    const keep = (from: PromoBackend, key: string, cards: readonly TeaserCard[]) => {
        if (from === backend) set((store) => ({cards: {...store.cards, [key]: cards}}))
    }

    const read = (key: string) => {
        const from = backend
        if (!from || reading.has(key)) return
        reading.add(key)
        from.teaser(key).then(
            (cards) => keep(from, key, cards),
            (e: unknown) => {
                console.error("Could not read the battle's teaser", e)
                keep(from, key, [])
            },
        )
    }

    return {
        cards: {},
        ask: (key) => {
            if (key === "") return
            asked.add(key)
            read(key)
        },
        serve: (next) => {
            backend = next
            reading.clear()
            if (next === undefined) {
                asked.clear()
                set({cards: {}})
                return
            }
            for (const key of asked) read(key)
        },
    }
})

// A battle's cards by its key: undefined until they are read, and for no key.
export function useBattleCards(key: string | undefined): readonly TeaserCard[] | undefined {
    const cards = useTeaserCards((store) => key === undefined ? undefined : store.cards[key])
    useEffect(() => {
        if (key !== undefined) useTeaserCards.getState().ask(key)
    }, [key])
    return cards
}
