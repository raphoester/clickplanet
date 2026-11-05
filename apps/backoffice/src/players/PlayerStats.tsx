import {ReactNode} from "react"
import {When} from "../components/When.tsx"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {spanOf} from "./durations.ts"

// A row of the ledger's 72 hours. Many takes and few tiles held is a player painted over as fast as it paints.
export function PlayerStats({player}: {player: Player}) {
    return (
        <dl className="stats">
            <Stat label="Takes">{player.takes}</Stat>
            <Stat label="Held">{player.tiles}{player.takes > 0 && <span className="muted"> ({shareOf(player)})</span>}</Stat>
            <Stat label="Takes/min">{player.takesPerMinute.toFixed(1)}</Stat>
            <Stat label="Tiles/min">{player.tilesPerMinute.toFixed(1)}</Stat>
            <Stat label="Active">{spanOf(player.activeFor)}</Stat>
        </dl>
    )
}

export function BanLine({player}: {player: Player}) {
    if (player.banned !== true) return null
    return (
        <span className="decision">
            <span className="badge badge-bug">Banned</span>
            <span className="muted">until</span>
            <When at={player.bannedUntil}/>
        </span>
    )
}

function shareOf(player: Player): string {
    const share = (100 * player.tiles) / player.takes
    return `${share < 10 ? share.toFixed(1) : Math.round(share)}% of takes`
}

function Stat({label, children}: {label: string, children: ReactNode}) {
    return (
        <div>
            <dt>{label}</dt>
            <dd>{children}</dd>
        </div>
    )
}
