import {PromoBackend, TeaserCard} from "./promo.ts"

// Every battle is teased with the same cards: made-up backgrounds drawn in the page, and words as the server fills
// them. `fakePromo.cards = []` on the console teases nothing.
export class FakePromoBackend implements PromoBackend {
    cards: TeaserCard[] = FAKE_CARDS
    readonly asked: string[] = []

    public async teaser(battleId: string): Promise<TeaserCard[]> {
        this.asked.push(battleId)
        return this.cards
    }
}

function svgOf(top: string, bottom: string, shapes: string): string {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 400"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1">` +
        `<stop offset="0" stop-color="${top}"/><stop offset="1" stop-color="${bottom}"/></linearGradient></defs>` +
        `<rect width="300" height="400" fill="url(#g)"/>${shapes}</svg>`
    return `data:image/svg+xml,${encodeURIComponent(svg)}`
}

const FAKE_CARDS: TeaserCard[] = [
    {
        backgroundUrl: svgOf("#1d2b53", "#7e2553", `<circle cx="150" cy="270" r="90" fill="#ff004d" opacity=".7"/><circle cx="90" cy="310" r="40" fill="#ffa300"/>`),
        headline: "Clicks refill 3× faster", line: "Every country fights at once.",
    },
    {
        backgroundUrl: svgOf("#0b3d2e", "#1d2b53", `<rect x="60" y="230" width="50" height="120" fill="#ffec27"/><rect x="125" y="190" width="50" height="160" fill="#ffec27"/><rect x="190" y="260" width="50" height="90" fill="#ffec27"/>`),
        headline: "A box every 1 min", line: "Keep your bombs and shields for it.",
    },
    {
        backgroundUrl: svgOf("#5f574f", "#000000", `<path d="M150 180 L180 270 L270 270 L195 320 L225 400 L150 350 L75 400 L105 320 L30 270 L120 270 Z" fill="#ff77a8"/>`),
        headline: "Bring your friends", line: "Your flag needs every click.",
    },
]
