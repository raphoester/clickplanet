import {Country} from "../domain/countries.ts";
import {ChevronIcon} from "./components/icons.tsx";
import "./MenuHeader.css"

export type MenuHeaderProps = {
    country: Country,
    /** 1-based, or null when the country holds no tile yet. */
    rank: number | null,
    isOpen: boolean,
    onToggle: () => void,
    /**
     * The region this header's button expands. Only referenced while it is on
     * screen: the body is unmounted when folded — a phone opens folded, and a
     * hidden leaderboard would still re-render on every live update — and an
     * `aria-controls` pointing at nothing is worse than none at all.
     */
    bodyId: string,
}

/**
 * The strip that never folds away.
 *
 * Two things earn a permanent place on it. The rank, because it is the one
 * number a player wants at a glance and the card is folded most of the time on
 * a phone. And, once folded, the country itself — a bar showing only a wordmark
 * and a chevron would be branding with a lid on it, so the wordmark gives way
 * to the flag and name it belongs to.
 *
 * The chevron sits on a row that names what it collapses, which is what the old
 * bare "Close" button lacked.
 */
export default function MenuHeader(props: MenuHeaderProps) {
    return <div className="menu-header">
        <img alt="ClickPlanet logo"
             src="/static/logo.svg"
             className="menu-header-logo"
             width="42px"
             height="42px"/>

        {props.isOpen
            ? <h1>ClickPlanet</h1>
            : <span className="menu-header-country">{props.country.name}</span>}

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
