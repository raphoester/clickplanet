import {ReactNode} from "react"
import {PlayerLine} from "../../backends/player.ts"
import {StandingsBackend} from "../../backends/standings.ts"
import {Country} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import HeadingSelect, {HeadingChoice} from "../components/HeadingSelect.tsx"
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
    onOpenPlayer?: (player: PlayerLine) => void
    view: BoardView
    onView: (view: BoardView) => void
}

export type BoardViewsProps = BoardStandings & {
    country: Country
    countries: ReactNode
}

export default function BoardViews(props: BoardViewsProps) {
    const choices: HeadingChoice<BoardView>[] = [
        {value: "countries", name: "Countries", label: "Countries"},
        {value: "players", name: "Players", label: "Players"},
        {
            value: "country",
            name: props.country.name,
            label: <span><CountryFlag code={props.country.code}/>{props.country.name}</span>,
        },
    ]

    return <>
        <HeadingSelect label="Leaderboard" choices={choices} value={props.view} onChange={props.onView}/>

        {props.view === "countries" && props.countries}
        {(props.view === "players" || props.view === "country") &&
            <PlayerStandings backend={props.backend}
                             countryCode={props.view === "country" ? props.country.code : ""}
                             label={props.view === "country" ? `Players, ${props.country.name}` : "Players"}
                             caller={props.caller}
                             listenForClicks={props.listenForClicks}
                             onSignIn={props.onSignIn}
                             onOpenPlayer={props.onOpenPlayer}/>}
    </>
}
