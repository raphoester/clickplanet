// A card of a battle's teaser: a public background, and words already filled with what its rule set plays.
export type TeaserCard = {
    backgroundUrl: string
    headline: string
    line: string
}

export interface PromoBackend {
    // The teaser's cards of a battle, by its entry's id or, for a season's finale, its round. None for a battle whose
    // set has none, one the schedule no longer holds, or a server without promo.
    teaser(battleId: string): Promise<TeaserCard[]>
}
