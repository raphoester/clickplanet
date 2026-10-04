import {MySeason} from "../../backends/standings.ts"
import {StatTile, StatTiles} from "../components/StatTiles.tsx"
import {Caller} from "./useMySeason.ts"
import "./Standings.css"

export type YourSeasonProps = {
    mine?: MySeason
    rankedAmong: string
    caller: Caller
    onSignIn?: () => void
}

export default function YourSeason({mine, rankedAmong, caller, onSignIn}: YourSeasonProps) {
    if (!mine && !onSignIn) return null

    const ranked = mine !== undefined && caller.username !== undefined

    return <section className="your-season" aria-label="Your season">
        <StatTiles>
            <StatTile label="Tiles" value={mine ? String(mine.tiles) : "—"}/>
            {ranked && <StatTile label={rankedAmong} value={rankLabel(mine.rank)}/>}
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
