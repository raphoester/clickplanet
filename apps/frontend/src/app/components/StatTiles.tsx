import {ReactNode} from "react"
import "./StatTiles.css"

export function StatTiles({children}: {children: ReactNode}) {
    return <dl className="stat-tiles">{children}</dl>
}

export function StatTile({label, value}: {label: string, value: string}) {
    return <div className="stat-tile">
        <dt>{label}</dt>
        <dd>{value}</dd>
    </div>
}
