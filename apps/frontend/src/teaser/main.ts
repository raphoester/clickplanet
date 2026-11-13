import "../tokens.css"
import "../app/battle/Battle.css"
import "./teaser.css"
import {Teaser} from "../gen/grpc/promo/v1/admin_pb.ts"
import {teaserFrameOf} from "../domain/teaserFrame.ts"

// The page scripts/teaser/render.mjs takes a still of for each card: the teaser it is handed, as promo's ListTeasers
// answers it, a card at a time.
declare global {
    interface Window {
        teaser?: {count: number, show(index: number): Promise<void>}
        teaserFailed?: string
    }
}

function element<K extends keyof HTMLElementTagNameMap>(tag: K, className: string, text?: string): HTMLElementTagNameMap[K] {
    const made = document.createElement(tag)
    made.className = className
    if (text !== undefined) made.textContent = text
    return made
}

// Every background in the browser's cache before the first still, so no card is taken half drawn.
async function loaded(url: string): Promise<void> {
    const image = new Image()
    image.src = url
    await decoded(image, url)
}

// decode() rejects with the same words for a refused download as for a broken file.
async function decoded(image: HTMLImageElement, url: string): Promise<void> {
    try {
        await image.decode()
    } catch (error) {
        throw new Error(`the background ${url} did not load: ${error instanceof Error ? error.message : String(error)}`)
    }
}

async function main() {
    const answer = await fetch("/__teaser/teaser.json")
    if (!answer.ok) throw new Error(`no teaser: ${answer.status}`)
    const teaser = Teaser.fromJson(await answer.json(), {ignoreUnknownFields: true})
    if (teaser.cards.length === 0 || teaser.startsAt === undefined) throw new Error("a teaser with no card or no start")
    const frame = teaserFrameOf(teaser.startsAt.toDate(), teaser.endsSeason)

    const root = document.getElementById("teaser")!
    const head = element("header", "teaser-frame-head")
    head.append(
        element("div", teaser.endsSeason ? "teaser-frame-name teaser-frame-name--finale" : "teaser-frame-name", frame.name),
        element("div", "teaser-frame-day", frame.day),
        element("div", "teaser-frame-time", frame.time),
    )
    const card = element("figure", "teaser-card")
    const background = element("img", "")
    background.alt = ""
    const caption = element("figcaption", "")
    const headline = element("strong", "teaser-card-headline")
    const line = element("span", "teaser-card-line")
    caption.append(headline, line)
    card.append(background, caption)
    const dots = element("div", "teaser-frame-dots")
    const each = teaser.cards.map(() => element("span", "teaser-frame-dot"))
    dots.append(...each)
    const foot = element("footer", "teaser-frame-foot")
    foot.append(dots, element("div", "teaser-frame-link", "clickplanet.lol"))
    root.append(element("div", "teaser-frame-tape"), head, card, foot)

    await Promise.all(teaser.cards.map((written) => loaded(written.backgroundUrl)))
    await document.fonts.ready

    window.teaser = {
        count: teaser.cards.length,
        async show(index: number) {
            const shown = teaser.cards[index]
            background.src = shown.backgroundUrl
            headline.textContent = shown.headline
            line.textContent = shown.line
            line.hidden = shown.line === ""
            each.forEach((dot, i) => dot.dataset.shown = String(i === index))
            await decoded(background, shown.backgroundUrl)
        },
    }
}

main().catch((error: unknown) => {
    window.teaserFailed = error instanceof Error ? error.message : String(error)
})
