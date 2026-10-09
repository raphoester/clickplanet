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
    order?: ReactNode
    headless?: boolean
}

export type BoardHeadingProps = Pick<BoardStandings, "view" | "onView"> & {
    country: Country
    order?: ReactNode
}

export function BoardHeading({view, onView, country, order}: BoardHeadingProps) {
    const choices: HeadingChoice<BoardView>[] = [
        {value: "countries", name: "Countries", label: <span>Countries</span>},
        {value: "players", name: "Players", label: <span>Players</span>},
        {
            value: "country",
            name: country.name,
            label: <span><CountryFlag code={country.code}/>{country.name}</span>,
        },
    ]

    return <div className="board-heading">
        <HeadingSelect label="Leaderboard" choices={choices} value={view} onChange={onView}/>
        {view === "countries" && order}
    </div>
}

export default function BoardViews(props: BoardViewsProps) {
    return <>
        {!props.headless && <BoardHeading view={props.view} onView={props.onView} country={props.country} order={props.order}/>}

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
