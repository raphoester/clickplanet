import {ReactNode} from "react"
import {Season} from "../../backends/season.ts"
import {finaleWindow} from "../../domain/seasonClock.ts"
import BonusIcon from "../components/BonusIcon.tsx"

export default function SeasonFacts({season, finale}: {season: Season, finale: boolean}) {
    return <ul className="season-facts">
        {finale
            ? <Fact art={<BombArt/>} name="Power-ups for all" text="The climax of the season. Everybody gets tons of them."/>
            : <Fact art={<BombArt/>}
                    name="Final Battle"
                    when={finaleLine(season)}
                    text="The climax of the season. Everybody gets tons of power-ups."/>}
        <Fact art={<GroundArt/>}
              name="Win the day"
              text={finale
                  ? "The countries holding the most ground score triple points."
                  : "Each day, the countries that held the most ground score points. The Final Battle scores triple."}/>
        <Fact art={<TrophyArt/>} name="Win" text="The winning country gets a trophy in the Hall of Fame."/>
        <Fact art={<MedalArt/>} name="Titles" text="Every signed-in player gets a title, and one more if their country wins."/>
    </ul>
}

function Fact({art, name, when, text}: {art: ReactNode, name: string, when?: string, text: string}) {
    return <li className="season-fact">
        {art}
        <span className="season-fact-body">
            <span className="season-fact-name">{name}</span>
            {when && <span className="season-fact-when">{when}</span>}
            <span className="season-fact-text">{text}</span>
        </span>
    </li>
}

function finaleLine(season: Season): string {
    const finale = finaleWindow(season)
    return `${finale.day} · ${finale.from}–${finale.to}`
}

function BombArt() {
    return <span className="season-fact-art season-fact-art--bomb"><BonusIcon kind="bomb"/></span>
}

function GroundArt() {
    return <span className="season-fact-art">
        <svg className="season-art" viewBox="0 0 48 48" aria-hidden="true">
            <polygon className="season-art-tile" points="24,18 36.5,25 36.5,39 24,46 11.5,39 11.5,25"/>
            <path className="season-art-pole" d="M24 32V3"/>
            <path className="season-art-flag" d="M25.5 4h15l-4 5.5 4 5.5h-15z"/>
        </svg>
    </span>
}

function TrophyArt() {
    return <span className="season-fact-art">
        <svg className="season-art" viewBox="0 0 48 48" aria-hidden="true">
            <path className="season-art-handle-ink" d="M33 10h6a4 4 0 0 1 0 8h-6M15 10H9a4 4 0 0 0 0 8h6"/>
            <path className="season-art-handle" d="M33 10h6a4 4 0 0 1 0 8h-6M15 10H9a4 4 0 0 0 0 8h6"/>
            <path className="season-art-gold" d="M14 6h20v10a10 10 0 0 1-20 0z"/>
            <path className="season-art-gold-shade" d="M21 27h6v7h-6z"/>
            <path className="season-art-gold" d="M15 34h18v7H15z"/>
            <path className="season-art-shine" d="M19 9v7"/>
        </svg>
    </span>
}

function MedalArt() {
    return <span className="season-fact-art">
        <svg className="season-art" viewBox="0 0 48 48" aria-hidden="true">
            <path className="season-art-gold-shade" d="M14 3h8l5 14h-8z"/>
            <path className="season-art-gold-shade" d="M34 3h-8l-5 14h8z"/>
            <circle className="season-art-gold" cx="24" cy="30" r="13"/>
            <circle className="season-art-well" cx="24" cy="30" r="8.5"/>
            <polygon className="season-art-star"
                     points="24,24 25.8,28.2 30.2,28.4 26.8,31.2 28,35.5 24,33 20,35.5 21.2,31.2 17.8,28.4 22.2,28.2"/>
        </svg>
    </span>
}
