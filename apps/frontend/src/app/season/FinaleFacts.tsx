import {Finale, finaleLengthLine, finaleWhenLine} from "../../domain/battle.ts"
import {SwordsIcon} from "../components/icons.tsx"
import {Fact, TrophyArt} from "./SeasonFacts.tsx"

// The season's Final Battle, from the schedule: when, how long, what it is worth, and the operator's words.
export function FinaleDetails({finale}: {finale: Finale}) {
    return <div className="finale-details">
        <p className="finale-details-head">
            <SwordsIcon/>
            <span className="finale-details-name">The Final Battle</span>
        </p>
        <p className="finale-details-when">{finaleWhenLine(finale)}</p>
        <FinaleFacts finale={finale}/>
    </div>
}

export function FinaleFacts({finale}: {finale: Finale}) {
    return <>
        {finale.description && <p className="finale-details-words">{finale.description}</p>}
        <ul className="season-facts">
            <Fact art={<HourglassArt/>}
                  name={`${finaleLengthLine(finale)} of war`}
                  text="The end of the season, with every country fighting at once."/>
            {finale.times > 1 && <Fact art={<SwordsArt/>}
                                       name={`Points ×${finale.times}`}
                                       text={`The countries holding the most ground when it ends score the points of ${finale.times} days.`}/>}
            <Fact art={<TrophyArt/>} name="The last word" text="Nothing scores after it. It crowns the season's champion."/>
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

function SwordsArt() {
    return <span className="season-fact-art">
        <svg className="season-art" viewBox="0 0 48 48" aria-hidden="true">
            <path className="finale-art-ink" d="M8 8l24 24M40 8L16 32M8 40l8-8M40 40l-8-8M6 26l8 8M42 26l-8 8"/>
            <path className="finale-art-blade" d="M8 8l24 24M40 8L16 32"/>
            <path className="finale-art-hilt" d="M8 40l8-8M40 40l-8-8M6 26l8 8M42 26l-8 8"/>
        </svg>
    </span>
}
