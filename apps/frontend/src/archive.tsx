import {createRoot} from "react-dom/client"
import {StrictMode} from "react"
import "./tokens.css"
import "./index.css"

import {loadBorders} from "./app/viewer/borderField.ts"
import {BORDERS_URL} from "./app/viewer/bordersAsset.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import SeasonMap from "./app/archive/SeasonMap.tsx"
import {FakeTurnBackend, SEASON_ZERO} from "./backends/fakeTurnBackend.ts"
import {TurnBackend} from "./backends/turn.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"

const borders = loadBorders(BORDERS_URL)
const geometry = () => Promise.all([loadPointGeometryData(), borders]).then(([{positions}, {assignment, codes}]) => ({
    positions, landmasses: {assignment, count: codes.length},
}))

function turnBackend(): TurnBackend {
    if (!(import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND)) {
        return new ConnectTurnBackend(newGamemasterServiceClient({baseUrl: API_BASE_URL, timeoutMs: 10_000}))
    }
    const fake = new FakeTurnBackend(undefined, Date.now, [SEASON_ZERO])
    fake.replayOn(() => borders.then(({assignment, codes}) => new Map(
        [...assignment].flatMap((land, i) => land && codes[land] ? [[i + 1, codes[land]] as const] : []),
    )))
    return fake
}

createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <SeasonMap turn={turnBackend()} geometry={geometry}/>
    </StrictMode>,
)
