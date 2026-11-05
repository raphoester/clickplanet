import {ReactNode} from "react"

// What a season is, the same every day: no battle is promised here, since the schedule decides them.
export default function SeasonFacts() {
    return <ul className="season-facts">
        <Fact art={<GroundArt/>}
              name="Win the day"
              text="When a day ends, the countries holding the most ground score points."/>
        <Fact art={<TrophyArt/>} name="Win" text="The winning country gets a trophy in the Hall of Fame."/>
        <Fact art={<MedalArt/>} name="Titles" text="Every signed-in player gets a title, and one more if their country wins."/>
    </ul>
}

export function Fact({art, name, text}: {art: ReactNode, name: string, text: string}) {
    return <li className="season-fact">
        {art}
        <span className="season-fact-body">
            <span className="season-fact-name">{name}</span>
            <span className="season-fact-text">{text}</span>
        </span>
    </li>
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

export function TrophyArt() {
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
