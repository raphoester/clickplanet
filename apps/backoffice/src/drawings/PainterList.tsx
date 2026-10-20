import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {PlayerName} from "../players/PlayerName.tsx"
import {countOf} from "../support/labels.ts"
import {Painters} from "./actions.ts"

// Who painted the area, most takes first, each opening on the player page, where the ban is.
export function PainterList({painters}: {painters: Painters}) {
    if (painters.painters.length === 0) return <p className="muted">Nobody painted it since then.</p>

    const shown = painters.painters.length
    return (
        <>
            <p className="muted">
                {shown < painters.paintersTotal
                    ? `${shown} of ${painters.paintersTotal} painters`
                    : countOf(painters.paintersTotal, "painter")}
            </p>
            {painters.unnamed !== undefined && <p className="action-refused">The names could not be read. {painters.unnamed}</p>}
            <ol className="painters" aria-label="Painters">
                {painters.painters.map(row => (
                    <Painter key={`${row.accountId}|${row.scope}`} row={row} name={painters.authors.get(row.accountId)?.name ?? ""}/>
                ))}
            </ol>
        </>
    )
}

function Painter({row, name}: {row: Player, name: string}) {
    const openPlayer = useOpenPlayer()
    const caller = row.accountId !== "" ? {accountId: row.accountId} : {scope: row.scope}

    return (
        <li className="painter">
            <PlayerName name={name} flag={row.flagCountryId} unnamed={row.accountId === "" ? "No account" : "Unnamed"}/>
            <span className="muted">{countOf(row.takes, "take")}, {row.tiles} held</span>
            {openPlayer !== undefined && (
                <button type="button" className="button-small" onClick={() => openPlayer({caller, name, row})}>Open</button>
            )}
        </li>
    )
}
