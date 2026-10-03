import {ReactNode} from "react"
import {StandingsBackend} from "../../backends/standings.ts"
import {Country} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import PlayerStandings from "./PlayerStandings.tsx"
import {Caller} from "./useMySeason.ts"
import "./Standings.css"

export type BoardView = "countries" | "players" | "country"

export type BoardStandings = {
    backend: StandingsBackend
    caller: Caller
    listenForClicks: ListenForClicks
    onSignIn?: () => void
    view: BoardView
    onView: (view: BoardView) => void
}

export type BoardViewsProps = BoardStandings & {
    country: Country
    countries: ReactNode
}

export default function BoardViews(props: BoardViewsProps) {
    const tab = (view: BoardView, label: ReactNode) => <button type="button"
                                                               role="tab"
                                                               aria-selected={props.view === view}
                                                               className={`button button-mini board-view${props.view === view ? " button-secondary" : ""}`}
                                                               onClick={() => props.onView(view)}>
        {label}
    </button>

    return <>
        <div className="board-views" role="tablist" aria-label="Leaderboard">
            {tab("countries", "Countries")}
            {tab("players", "Players")}
            {tab("country", <>
                <CountryFlag code={props.country.code}/>
                <span className="board-view-name">{props.country.name}</span>
            </>)}
        </div>

        {props.view === "countries"
            ? props.countries
            : <PlayerStandings backend={props.backend}
                               countryCode={props.view === "country" ? props.country.code : ""}
                               label={props.view === "country" ? `Players, ${props.country.name}` : "Players"}
                               caller={props.caller}
                               listenForClicks={props.listenForClicks}
                               onSignIn={props.onSignIn}/>}
    </>
}
