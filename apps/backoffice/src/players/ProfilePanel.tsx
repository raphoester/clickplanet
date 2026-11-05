import {useState} from "react"
import {Country} from "../components/Country.tsx"
import {Fact} from "../components/Fact.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {AdminGetPlayerResponse} from "../gen/grpc/player/v1/admin_pb.ts"
import {NameColor} from "../gen/grpc/player/v1/color_pb.ts"
import {AdminGetProgressResponse} from "../gen/grpc/progression/v1/admin_pb.ts"
import {CountryTiles} from "../gen/grpc/progression/v1/progression_pb.ts"
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

function Facts({profile: {identity, progress}}: {profile: Profile}) {
    const created = identity !== undefined && identity.createdAtUnixMs !== 0n ? new Date(Number(identity.createdAtUnixMs)) : undefined
    const streak = progress.stats?.streakCurrent ?? 0
    const best = progress.stats?.streakBest ?? 0

    return (
        <dl className="facts">
            {identity !== undefined && (
                <Fact label="Account">
                    {identity.linked ? "Signed in" : "Guest"}
                    {created !== undefined && <>, made <When at={created}/></>}
                </Fact>
            )}
            <Fact label="Name"><NameLine identity={identity}/></Fact>
            {identity?.admin === true && <Fact label="Admin">Yes: the game draws a crown beside its name</Fact>}
            {identity !== undefined && !identity.guest && <Fact label="Color">{colorLabel(identity.color)}</Fact>}
            <Fact label="Tiles taken">{String(progress.stats?.tilesTaken ?? 0n)}</Fact>
            <Fact label="Streak">{countOf(streak, "day")} now, best {countOf(best, "day")}</Fact>
            <Fact label="Messages">{String(progress.messagesSent)}</Fact>
            <Fact label="Titles">{titlesLine(progress)}</Fact>
            <Fact label="Plays for"><Fronts countries={progress.playsFor}/></Fact>
            <Fact label="Plays against"><Fronts countries={progress.playsAgainst}/></Fact>
        </dl>
    )
}

function NameLine({identity}: {identity: AdminGetPlayerResponse | undefined}) {
    if (identity === undefined || identity.name === "") {
        return <span className="muted">none: it never joined the roster or posted, so it has no guest code</span>
    }
    return (
        <>
            {identity.name} <span className="muted">{identity.guest ? "(guest code: no username)" : "(username)"}</span>
        </>
    )
}

function titlesLine(progress: AdminGetProgressResponse): string {
    if (progress.titles.length === 0) return "none"
    return progress.titles.map(title => title.id === progress.wornTitle?.id ? `${title.name} (worn)` : title.name).join(", ")
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
