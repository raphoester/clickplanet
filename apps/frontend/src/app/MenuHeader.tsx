import {Country, nameWithoutFlag} from "../domain/countries.ts";
import {ChevronIcon} from "./components/icons.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import "./MenuHeader.css"

export type MenuHeaderProps = {
    country: Country,
    rank: number | null,
    isOpen: boolean,
    onToggle: () => void,
    bodyId: string,
}

export default function MenuHeader(props: MenuHeaderProps) {
    return <div className="menu-header">
        <img alt="ClickPlanet logo"
             src="/static/logo.svg"
             className="menu-header-logo"
             width="42px"
             height="42px"/>

        {props.isOpen
            ? <h1>ClickPlanet</h1>
            : <span className="menu-header-country">
                <CountryFlag code={props.country.code}/>
                {nameWithoutFlag(props.country)}
            </span>}

        <span className="menu-header-spacer"/>

        <p className="menu-header-rank">
            <span className="menu-label">Rank</span>
            <span className="menu-rank-value">{props.rank === null ? "—" : `#${props.rank}`}</span>
        </p>

        <button type="button"
                className="icon-button menu-collapse"
                aria-label="ClickPlanet menu"
                aria-expanded={props.isOpen}
                aria-controls={props.isOpen ? props.bodyId : undefined}
                onClick={props.onToggle}>
            <ChevronIcon/>
        </button>
    </div>
}
