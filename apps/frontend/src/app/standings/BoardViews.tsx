import {ReactNode} from "react"
import {Boards} from "../../backends/standings.ts"
import {Country} from "../../domain/countries.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import HeadingSelect, {HeadingChoice} from "../components/HeadingSelect.tsx"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import PlayerStandings from "./PlayerStandings.tsx"
import {Caller} from "./useMySeason.ts"
import "./Standings.css"

export type BoardView = "countries" | "players" | "country"

export type BoardStandings = {
    backend: Boards
    caller: Caller
    listenForClicks: ListenForClicks
    onSignIn?: () => void
    view: BoardView
    onView: (view: BoardView) => void
}

export type BoardViewsProps = BoardStandings & {
    country: Country
    countries: ReactNode
    headless?: boolean
}

export type BoardHeadingProps = Pick<BoardStandings, "view" | "onView"> & {
    country: Country
}

export function BoardHeading({view, onView, country}: BoardHeadingProps) {
    const name = translator().country(country.code)
    const choices: HeadingChoice<BoardView>[] = [
        {value: "countries", name: t("standings.countries"), label: <span>{t("standings.countries")}</span>},
        {value: "players", name: t("standings.players"), label: <span>{t("standings.players")}</span>},
        {
            value: "country",
            name,
            label: <span><CountryFlag code={country.code}/>{name}</span>,
        },
    ]

    return <div className="board-heading">
        <HeadingSelect label={t("standings.leaderboard")} choices={choices} value={view} onChange={onView}/>
    </div>
}

export default function BoardViews(props: BoardViewsProps) {
    return <>
        {!props.headless && <BoardHeading view={props.view} onView={props.onView} country={props.country}/>}

        {props.view === "countries" && props.countries}
        {(props.view === "players" || props.view === "country") &&
            <PlayerStandings backend={props.backend}
                             countryCode={props.view === "country" ? props.country.code : ""}
                             label={props.view === "country"
                                 ? t("standings.playersOf", {country: translator().country(props.country.code)})
                                 : t("standings.players")}
                             caller={props.caller}
                             listenForClicks={props.listenForClicks}
                             onSignIn={props.onSignIn}/>}
    </>
}
