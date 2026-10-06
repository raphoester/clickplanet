const PLAY_PATH = "/play"

export function carryQuery(root: ParentNode, search: string) {
    if (!search) return
    for (const link of root.querySelectorAll(`a[href="${PLAY_PATH}"]`)) link.setAttribute("href", PLAY_PATH + search)
}
