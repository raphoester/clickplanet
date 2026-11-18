import {useCallback, useState} from "react"
import {Places} from "../boards/places.ts"
import {Clients} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {Country} from "../components/Country.tsx"
import {Fact} from "../components/Fact.tsx"
import {Ago, When} from "../components/When.tsx"
import {Graffiti} from "../gen/grpc/vandalsquad/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {readingOf} from "../players/actions.ts"
import {whoOf} from "../players/caller.ts"
import {BanAction} from "../players/Moderation.tsx"
import {RevertPanel} from "../players/RevertPanel.tsx"
import {Standing} from "../players/Standing.tsx"
import {Call, useCall} from "../useCall.ts"
import {GraffitiPicture} from "./GraffitiPicture.tsx"
import {keyOf, tilesOf} from "./graffiti.ts"

type Props = {
    graffiti: Graffiti
    // The painter's name, or "" when the page has none.
    name: string
    // The painter's other graffiti, newest first.
    others: Graffiti[]
    // The places of the graffiti's board.
    places: Call<Places>
    clients: Clients
    onBack: () => void
    onOpen: (key: string) => void
    // A ban changed where the painter stands.
    onBanned: () => void
}

// One graffiti, read top down as it is judged: the drawing, who painted it and where it stands, then the ban and the
// revert of its painter.
export function GraffitiDetail({graffiti, name, others, places, clients, onBack, onOpen, onBanned}: Props) {
    const accountId = graffiti.accountId
    const caller = {accountId}
    const who = whoOf(caller, name)
    const readStanding = useCallback(
        (signal: AbortSignal) => readingOf(clients.antibot, {accountId}, signal),
        [clients.antibot, accountId],
    )
    const [standing, readStandingAgain] = useCall(readStanding)
    const banned = () => {
        readStandingAgain()
        onBanned()
    }

    return (
        <section className="detail graffiti-detail" aria-label="Graffiti">
            <button type="button" className="button-small back" onClick={onBack}>← All graffiti</button>
            <article className="card">
                <div className="card-head">
                    <span className="badge badge-report">Swastika</span>
                    <When at={graffiti.flaggedAt}/>
                </div>
                <figure className="graffiti-figure">
                    {places.state === "loading" && <p className="muted">Loading the map…</p>}
                    {places.state === "failed" && (
                        <p className="muted">Its tiles could not be placed: {places.detail}</p>
                    )}
                    {places.state === "done" && (
                        <GraffitiPicture
                            places={places.value}
                            tileIds={graffiti.tileIds}
                            label={`The ${tilesOf(graffiti)} ${who} painted, north up, over the land around them`}
                        />
                    )}
                    <figcaption className="hint">
                        Its tiles as the squad last saw them, north up, over the land around them; the sea is blank.
                    </figcaption>
                </figure>
                <dl className="facts">
                    <Fact label="Painter"><Painter accountId={accountId} name={name}/></Fact>
                    <Fact label="Flag"><Country id={graffiti.countryId}/></Fact>
                    <Fact label="Size">{tilesOf(graffiti)}</Fact>
                    <Fact label="Board">{boardOf(graffiti)}</Fact>
                    <Fact label="Flagged"><When at={graffiti.flaggedAt}/></Fact>
                    <Fact label="Last seen"><When at={graffiti.seenAt}/></Fact>
                    <Fact label="On the map">
                        {graffiti.link === ""
                            ? <span className="muted">no link</span>
                            : <a href={graffiti.link} target="_blank" rel="noreferrer">Open on the globe</a>}
                    </Fact>
                </dl>
                {accountId !== "" && <div className="standing-line"><Standing reading={standing}/></div>}
                {others.length > 0 && <Others graffiti={others} onOpen={onOpen}/>}
            </article>
            {accountId === ""
                ? <p className="muted">Its painter deleted its account: there is nobody left to ban or revert.</p>
                : <Tools clients={clients} accountId={accountId} who={who} onBanned={banned}/>}
        </section>
    )
}

// The board its tiles are on, by its name: a tile id is a place on it alone.
function boardOf(graffiti: Graffiti): string {
    return graffiti.board === "" ? "No board" : graffiti.board
}

function Painter({accountId, name}: {accountId: string, name: string}) {
    const openPlayer = useOpenPlayer()
    if (accountId === "") return <span className="muted">a deleted account</span>
    return (
        <span className="painter">
            {openPlayer === undefined
                ? <span className={name === "" ? "muted" : "name"}>{name === "" ? "Unnamed" : name}</span>
                : (
                    <button
                        type="button"
                        className="line-author"
                        title="Open on the player page, with every tool"
                        onClick={() => openPlayer({caller: {accountId}, name})}
                    >
                        {name === "" ? "Unnamed" : name}
                    </button>
                )}
            <Copyable text={accountId} short/>
        </span>
    )
}

// The painter's other graffiti: one click to each.
function Others({graffiti, onOpen}: {graffiti: Graffiti[], onOpen: (key: string) => void}) {
    return (
        <div className="related">
            <span className="muted">{graffiti.length === 1 ? "One more by this painter:" : `${graffiti.length} more by this painter:`}</span>
            <ul className="chips">
                {graffiti.map(each => (
                    <li key={keyOf(each)}>
                        <button type="button" className="chip" onClick={() => onOpen(keyOf(each))}>
                            {/* A chip is a flex box, which drops the spaces around a lone dot. */}
                            {tilesOf(each)}{"\u00a0·\u00a0"}<Ago at={each.flaggedAt}/>
                        </button>
                    </li>
                ))}
            </ul>
        </div>
    )
}

type Tool = "ban" | "revert"

// Ban first: an unbanned painter paints again behind the revert.
function Tools({clients, accountId, who, onBanned}: {clients: Clients, accountId: string, who: string, onBanned: () => void}) {
    const [tool, setTool] = useState<Tool>("ban")
    const tools: {key: Tool, label: string}[] = [{key: "ban", label: "Ban"}, {key: "revert", label: "Revert"}]
    return (
        <section className="panel tools" aria-label="Tools">
            <div className="segmented" role="tablist" aria-label="Tool">
                {tools.map(each => (
                    <button
                        key={each.key}
                        type="button"
                        role="tab"
                        aria-selected={tool === each.key}
                        onClick={() => setTool(each.key)}
                    >
                        {each.label}
                    </button>
                ))}
            </div>
            <div role="tabpanel" aria-label={tools.find(each => each.key === tool)?.label}>
                {tool === "ban" && (
                    <>
                        <p className="hint">
                            The account is banned alone, from any address. Its name opens the player page, with every
                            other tool.
                        </p>
                        <BanAction antibot={clients.antibot} caller={{accountId}} who={who} onDone={onBanned}/>
                    </>
                )}
                {tool === "revert" && <RevertPanel planet={clients.planet} caller={{accountId}} who={who}/>}
            </div>
        </section>
    )
}
