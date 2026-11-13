import {useCallback} from "react"
import {putFile} from "../api.ts"
import {BonusesAdmin, PlanetAdmin, PromoAdmin, QuizzesAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {useCall} from "../useCall.ts"
import {Put} from "./actions.ts"
import {BackgroundsPanel} from "./BackgroundsPanel.tsx"
import {CardsPanel} from "./CardsPanel.tsx"

// The battle teasers: the background library, then each rule set's cards over it, as players will read them.
export function PromoPage({promo, planet, bonuses, quizzes, put = putFile}: {
    promo: PromoAdmin, planet: PlanetAdmin, bonuses: BonusesAdmin, quizzes: QuizzesAdmin, put?: Put,
}) {
    const listBackgrounds = useCallback(
        async (signal: AbortSignal) => (await promo.listBackgrounds({}, {signal})).backgrounds,
        [promo],
    )
    const [library, againLibrary] = useCall(listBackgrounds)

    // The cards as the server keeps and fills them, and planet's sets, the offers' and the quizzes', so a card being
    // written fills as it is typed: `{bonuses.…}` from the offers', `{quizzes.…}` from the quizzes', the rest from planet's.
    const readCards = useCallback(async (signal: AbortSignal) => {
        const [cards, rules, offers, asked] = await Promise.all([
            promo.getCards({}, {signal}), planet.getRules({}, {signal}), bonuses.getRules({}, {signal}),
            quizzes.getRules({}, {signal}),
        ])
        return {cards, rules, prefixed: {bonuses: offers, quizzes: asked}}
    }, [promo, planet, bonuses, quizzes])
    const [cards, againCards] = useCall(readCards)

    return (
        <div className="promo">
            <p className="muted">
                A battle is teased in the game with its rule set's cards: a background and words over it. A{" "}
                <code>{"{path}"}</code> in the words is a value the set plays, filled when a player reads them.
            </p>
            <BackgroundsPanel promo={promo} put={put} library={library} onChanged={againLibrary}/>
            {cards.state === "loading" && <p className="muted">Loading the cards…</p>}
            {cards.state === "failed" && <FailureNotice failure={cards} onRetry={againCards}/>}
            {cards.state === "done" && (
                <CardsPanel
                    promo={promo}
                    library={library.state === "done" ? library.value : []}
                    cards={cards.value.cards}
                    rules={cards.value.rules}
                    prefixed={cards.value.prefixed}
                    onSaved={againCards}
                />
            )}
        </div>
    )
}
