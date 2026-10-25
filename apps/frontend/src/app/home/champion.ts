import {EndedSeason} from "../../backends/hallOfFame.ts"
import {Countries} from "../../domain/countries.ts"
import {championsOf} from "../../domain/hallOfFame.ts"

export function showChampion(root: ParentNode, hall: readonly EndedSeason[]) {
    const latest = hall[0]
    const champions = latest ? championsOf(latest) : []
    if (champions.length === 0) return

    write(root, "champion-season", String(latest.number))
    write(root, "champion-country", champions.map((champion) => Countries.get(champion.countryCode)?.name ?? champion.countryCode).join(" & "))
    for (const element of root.querySelectorAll<HTMLElement>("[data-champion]")) element.hidden = false
}

function write(root: ParentNode, field: string, text: string) {
    for (const element of root.querySelectorAll(`[data-${field}]`)) element.textContent = text
}
