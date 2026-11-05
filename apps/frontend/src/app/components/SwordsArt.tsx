import "./SwordsArt.css"

// Two crossed swords, flat and inked like the medals: for a disc or an art box, where the line icon looks like sticks.
export default function SwordsArt() {
    return <svg className="swords-art" viewBox="0 0 48 48" aria-hidden="true">
        <g transform="matrix(-1 0 0 1 48 0)"><Sword/></g>
        <Sword/>
    </svg>
}

function Sword() {
    return <>
        <polygon className="swords-art-blade" points="29,35 8,14 6,6 14,8 35,29"/>
        <path className="swords-art-ink" d="M26 38l12-12M32 32l7 7"/>
        <path className="swords-art-grip" d="M32 32l7 7"/>
        <path className="swords-art-guard" d="M26 38l12-12"/>
        <circle className="swords-art-pommel" cx="40" cy="40" r="3.2"/>
    </>
}
