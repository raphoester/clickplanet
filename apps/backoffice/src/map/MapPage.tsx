import {Clients, CLIENTS} from "../clients.ts"
import {PaintPanel} from "./PaintPanel.tsx"
import {ReassignPanel} from "./ReassignPanel.tsx"
import {ReplayPanel} from "./ReplayPanel.tsx"

// The tools on the map itself, not on one player's part of it. A reassign and a paint change it while the game
// runs, a tile at a time on the live stream, and each counts first; a replay reads it.
export function MapPage({clients = CLIENTS}: {clients?: Clients}) {
    return (
        <div className="detail">
            <ReassignPanel planet={clients.planet}/>
            <PaintPanel planet={clients.planet}/>
            <ReplayPanel planet={clients.planet}/>
        </div>
    )
}
