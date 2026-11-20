import {codeOf, Entry, LIST_LENGTH, ListName} from "../domain/clip/lists.ts"
import {DISPLAY_FONT_DRAWS, nameOf, said} from "./names.ts"

export type Rank = {code: string, line: string}

// The cover over the globe, the ranks from the 5th to the 1st, and the address on the last photo alone.
export type Photo =
    | {kind: "cover", hook: string}
    | {kind: "rank", rank: number, code: string, name: string, numbers: string[], line: string}
    | {kind: "last"}

const NUMBER = new Intl.NumberFormat("en-US")

// A share never more than it is: 81.9% is 81%, and under a tenth to a tenth of a point.
export function percentOf(share: number): string {
    return share >= 0.1 ? `${Math.floor(share * 100)}%` : `${(Math.floor(share * 1000) / 10).toFixed(1)}%`
}

// fight-the-most ranks by the sum of the two, and shows each on its line.
export function numbersOf(entry: Entry): string[] {
    if ("fought" in entry) return [`TOOK ${NUMBER.format(entry.took)}`, `LOST ${NUMBER.format(entry.lost)}`]
    if ("km" in entry) return [`${NUMBER.format(entry.km)} KM`]
    return [percentOf(entry.share)]
}

// The editor's ranks are the list's first 5, in its order: a pick that says otherwise is refused, the job checked it.
export function photosOf(list: ListName, entries: readonly Entry[], ranks: readonly Rank[], hook: string): Photo[] {
    if (ranks.length !== LIST_LENGTH || entries.length < LIST_LENGTH) {
        throw new Error(`a carousel tells ${LIST_LENGTH} ranks of a list of ${LIST_LENGTH}: ${ranks.length} of ${entries.length} in ${list}`)
    }
    const told = ranks.map(({code, line}, i): Photo => {
        if (codeOf(entries[i]) !== code) throw new Error(`rank ${i + 1} of ${list} is "${codeOf(entries[i])}", not "${code}"`)
        return {kind: "rank", rank: i + 1, code, name: said(nameOf(code)).toUpperCase(), numbers: numbersOf(entries[i]), line}
    })
    return [{kind: "cover", hook}, ...told.reverse(), {kind: "last"}]
}

export type Carousel = {
    show(index: number): void
    // Once the fonts and the flags are in: a name too long for its row is set smaller, never cut.
    fit(): void
}

function element<K extends keyof HTMLElementTagNameMap>(tag: K, className: string, parent: HTMLElement) {
    const created = document.createElement(tag)
    created.className = className
    parent.append(created)
    return created
}

export function createCarousel(root: HTMLElement, photos: readonly Photo[]): Carousel {
    const words = photos.flatMap((photo) => photo.kind === "cover" ? [photo.hook] : photo.kind === "rank" ? [photo.name, photo.line] : [])
    if (!words.every((text) => DISPLAY_FONT_DRAWS.test(text))) root.classList.add("clip--fallback-font")

    const names: HTMLElement[] = []
    const shown = photos.map((photo) => {
        const card = element("section", `carousel carousel--${photo.kind}`, root)
        if (photo.kind === "cover") {
            element("h1", "carousel-hook", card).textContent = photo.hook
        } else if (photo.kind === "rank") {
            const top = element("header", "carousel-top", card)
            element("p", "carousel-rank", top).textContent = `#${photo.rank}`
            const title = element("h1", "carousel-title", top)
            const flag = element("img", "carousel-flag", title)
            flag.src = `/static/countries/svg/${photo.code}.svg`
            flag.alt = ""
            const name = element("span", "carousel-name", title)
            name.textContent = photo.name
            names.push(name)
            const bottom = element("footer", "carousel-bottom", card)
            for (const number of photo.numbers) element("p", "carousel-number", bottom).textContent = number
            element("p", "carousel-line", bottom).textContent = photo.line
        } else {
            const mark = element("div", "carousel-mark", card)
            const logo = element("img", "carousel-logo", mark)
            logo.src = "/static/logo.svg"
            logo.alt = ""
            element("p", "carousel-site", mark).textContent = "clickplanet.lol"
        }
        return card
    })

    return {
        show(index) {
            shown.forEach((card, i) => card.style.display = i === index ? "" : "none")
        },
        fit() {
            for (const name of names) {
                const room = name.parentElement?.clientWidth ?? 0
                const drawn = name.parentElement?.scrollWidth ?? 0
                if (drawn > room && room > 0) name.style.fontSize = `${Math.floor(room / drawn * 100) / 100}em`
            }
        },
    }
}
