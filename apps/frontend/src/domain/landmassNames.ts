import generated from "../../static/landmassNames.json"
import {Countries} from "./countries.ts"

const names: Record<string, string> = generated.names

export const LANDMASS_NAMES_BORDERS = generated.borders

// A country's main landmass is named after the country, as this client names it.
export function landmassName(landmass: number, ground: string | undefined): string {
    const own = names[String(landmass)]
    if (own) return own
    return (ground && Countries.get(ground)?.name) ?? ground ?? ""
}
