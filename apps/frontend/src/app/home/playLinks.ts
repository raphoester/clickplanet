const PLAY_PATH = "/play"
const ANY_ORIGIN = "https://clickplanet.lol"

export function carryQuery(root: ParentNode, search: string) {
    if (!search) return
    for (const link of root.querySelectorAll(`a[href="${PLAY_PATH}"], a[href^="${PLAY_PATH}?"]`)) {
        const href = new URL(link.getAttribute("href") ?? PLAY_PATH, ANY_ORIGIN)
        for (const [name, value] of new URLSearchParams(search)) {
            if (!href.searchParams.has(name)) href.searchParams.append(name, value)
        }
        link.setAttribute("href", href.pathname + href.search)
    }
}
