import {ReactNode} from "react"
import {t} from "../../i18n/page.ts"
import {CrownIcon} from "../components/icons.tsx"
import type {MapShot} from "./mapShot.ts"
import "./FlagSubmission.css"

export type FlagCardProps = {
    name: string
    // The map at the capital with the flag on it, once the globe took it.
    shot?: MapShot
    // The picture alone, until then.
    picture?: string
    // The capital's place, or its coordinates.
    capital?: string
    chip?: ReactNode
}

// The flag as the players will see it: the game's own map around its capital, its name and the capital's place.
export default function FlagCard({name, shot, picture, capital, chip}: FlagCardProps) {
    return <div className="panel-box flag-card">
        {(shot || picture) && <span className="flag-card-map" aria-hidden="true">
            {shot
                ? <>
                    <img className="flag-card-shot" src={shot.url} alt="" style={{width: shot.width, height: shot.height}}/>
                    <span className="flag-card-crown" style={{left: `calc(50% + ${shot.capital.x}px)`, top: `calc(50% + ${shot.capital.y}px)`}}>
                        <CrownIcon size={22}/>
                    </span>
                </>
                : <img className="flag-card-picture" src={picture} alt=""/>}
        </span>}
        <span className="flag-card-text">
            <span className={name ? "flag-card-name" : "flag-card-name flag-card-name--empty"}>
                {name || t("flagSubmission.namePlaceholder")}
            </span>
            {capital && <span className="flag-card-capital"><CrownIcon size={14}/>{capital}</span>}
            {chip}
        </span>
    </div>
}
