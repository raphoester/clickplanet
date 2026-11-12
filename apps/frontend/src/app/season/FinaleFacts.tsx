import {ReactNode} from "react"
import {Finale, finaleMinutes} from "../../domain/battle.ts"
import {t, translator} from "../../i18n/page.ts"
import {SwordsIcon} from "../components/icons.tsx"
import SwordsArt from "../components/SwordsArt.tsx"
import {dayAndTimeRange} from "./dates.ts"
import {Fact, TrophyArt} from "./SeasonFacts.tsx"

// The season's Final Battle, from the schedule: when, how long, what it is worth, and the operator's words. `children`
// go under them: its rules, once it is the next battle.
export function FinaleDetails({finale, children}: {finale: Finale, children?: ReactNode}) {
    return <div className="finale-details">
        <p className="finale-details-head">
            <SwordsIcon/>
            <span className="finale-details-name">{t("finale.title")}</span>
        </p>
        <p className="finale-details-when">{dayAndTimeRange(finale.startsAt, finale.endsAt)}</p>
        <FinaleFacts finale={finale}/>
        {children}
    </div>
}

export function FinaleFacts({finale}: {finale: Finale}) {
    return <>
        {finale.description && <p className="finale-details-words">{finale.description}</p>}
        <ul className="season-facts">
            <Fact art={<HourglassArt/>}
                  name={t("finale.lengthOfWar", {length: translator().length(finaleMinutes(finale))})}
                  text={t("finale.lengthRule")}/>
            {finale.times > 1 && <Fact art={<span className="season-fact-art"><SwordsArt/></span>}
                                       name={t("battle.pointsTimes", {times: finale.times})}
                                       text={t("finale.timesRule", {times: finale.times})}/>}
            <Fact art={<TrophyArt/>} name={t("finale.lastWord")} text={t("finale.lastWordRule")}/>
        </ul>
    </>
}

function HourglassArt() {
    return <span className="season-fact-art">
        <svg className="season-art" viewBox="0 0 48 48" aria-hidden="true">
            <path className="finale-art-glass" d="M15 8h18c0 9-7 12-7 16s7 7 7 16H15c0-9 7-12 7-16s-7-7-7-16z"/>
            <path className="season-art-gold" d="M18.5 12h11c-1 4-5.5 6-5.5 8c0-2-4.5-4-5.5-8z"/>
            <path className="season-art-gold" d="M17 40c1-5 5-7.5 7-7.5s6 2.5 7 7.5z"/>
            <rect className="season-art-gold-shade" x="11" y="3" width="26" height="6" rx="2"/>
            <rect className="season-art-gold-shade" x="11" y="39" width="26" height="6" rx="2"/>
        </svg>
    </span>
}
