import {MySeason} from "../../backends/standings.ts"
import {Countries} from "../../domain/countries.ts"
import {StatTile, StatTiles} from "../components/StatTiles.tsx"
import {Caller} from "./useMySeason.ts"
import "./Standings.css"

export type YourSeasonProps = {
    mine?: MySeason
    caller: Caller
    onSignIn?: () => void
}

export default function YourSeason({mine, caller, onSignIn}: YourSeasonProps) {
    if (!mine && !onSignIn) return null

    const ranked = mine !== undefined && caller.username !== undefined
    const country = mine?.countryCode === undefined ? undefined : Countries.get(mine.countryCode)

    return <section className="your-season" aria-label="Your season">
        <StatTiles>
            <StatTile label="Tiles" value={mine ? String(mine.tiles) : "—"}/>
            {ranked && <StatTile label="Players" value={rankLabel(mine.globalRank)}/>}
            {ranked && country && <StatTile label={country.name} value={rankLabel(mine.countryRank)}/>}
        </StatTiles>
        {onSignIn && <button type="button"
                             className="button button-mini button-action your-season-sign-in"
                             onClick={onSignIn}>
            Sign in
        </button>}
    </section>
}

function rankLabel(rank: number | undefined): string {
    return rank === undefined ? "—" : `#${rank}`
}
