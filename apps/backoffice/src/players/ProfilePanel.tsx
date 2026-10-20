import {useState} from "react"
import {Country} from "../components/Country.tsx"
import {Fact} from "../components/Fact.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {NameColor} from "../gen/grpc/player/v1/color_pb.ts"
import {CountryTiles} from "../gen/grpc/player/v1/player_pb.ts"
import {countOf} from "../support/labels.ts"
import {Call} from "../useCall.ts"
import {Profile} from "./actions.ts"

// The fronts past these fold behind "All": a player who clicked everywhere plays for a hundred countries.
const SHOWN_FRONTS = 5

export const NO_PROFILE = "The game keeps nothing about this account: no name and no stats."

type Props = {profile: Call<Profile | undefined>, onReload: () => void}

// What the game keeps about an account: what its public card shows, and what a guest, which has no card, holds.
export function ProfilePanel({profile, onReload}: Props) {
    return (
        <section className="panel" aria-label="Profile">
            <div className="panel-head">
                <h2>Profile</h2>
                <button type="button" className="button-small" onClick={onReload}>Reload</button>
            </div>
            {profile.state === "loading" && <p className="muted">Loading…</p>}
            {profile.state === "failed" && <FailureNotice failure={profile} onRetry={onReload}/>}
            {profile.state === "done" && profile.value === undefined && <p className="muted">{NO_PROFILE}</p>}
            {profile.state === "done" && profile.value !== undefined && <Facts profile={profile.value}/>}
        </section>
    )
}

function Facts({profile}: {profile: Profile}) {
    const created = profile.createdAtUnixMs !== 0n ? new Date(Number(profile.createdAtUnixMs)) : undefined
    const streak = profile.stats?.streakCurrent ?? 0
    const best = profile.stats?.streakBest ?? 0

    return (
        <dl className="facts">
            <Fact label="Account">
                {profile.linked ? "Signed in" : "Guest"}
                {created !== undefined && <>, made <When at={created}/></>}
            </Fact>
            <Fact label="Name"><NameLine profile={profile}/></Fact>
            {profile.admin && <Fact label="Admin">Yes: the game draws a crown beside its name</Fact>}
            {!profile.guest && <Fact label="Color">{colorLabel(profile.color)}</Fact>}
            <Fact label="Tiles taken">{String(profile.stats?.tilesTaken ?? 0n)}</Fact>
            <Fact label="Streak">{countOf(streak, "day")} now, best {countOf(best, "day")}</Fact>
            <Fact label="Messages">{String(profile.messagesSent)}</Fact>
            <Fact label="Titles">{titlesLine(profile)}</Fact>
            <Fact label="Plays for"><Fronts countries={profile.playsFor}/></Fact>
            <Fact label="Plays against"><Fronts countries={profile.playsAgainst}/></Fact>
        </dl>
    )
}

function NameLine({profile}: {profile: Profile}) {
    if (profile.name === "") return <span className="muted">none: it never joined the roster or posted, so it has no guest code</span>
    return (
        <>
            {profile.name} <span className="muted">{profile.guest ? "(guest code: no username)" : "(username)"}</span>
        </>
    )
}

function titlesLine(profile: Profile): string {
    if (profile.titles.length === 0) return "none"
    return profile.titles.map(title => title.id === profile.wornTitle?.id ? `${title.name} (worn)` : title.name).join(", ")
}

function colorLabel(color: NameColor): string {
    if (color === NameColor.UNSPECIFIED) return "none: drawn grey"
    const word = NameColor[color].toLowerCase()
    return word.charAt(0).toUpperCase() + word.slice(1)
}

function Fronts({countries}: {countries: CountryTiles[]}) {
    const [all, setAll] = useState(false)
    if (countries.length === 0) return <span className="muted">nobody</span>

    const shown = all ? countries : countries.slice(0, SHOWN_FRONTS)
    return (
        <span className="fronts">
            {shown.map(country => (
                <span key={country.countryId} className="front">
                    <Country id={country.countryId}/>
                    <span className="muted">{String(country.tiles)}</span>
                </span>
            ))}
            {countries.length > SHOWN_FRONTS && (
                <button type="button" className="button-small" onClick={() => setAll(!all)}>
                    {all ? "Fewer" : `All ${countries.length}`}
                </button>
            )}
        </span>
    )
}
