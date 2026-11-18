import {createRoot} from "react-dom/client"
import {StrictMode} from "react"
import "./tokens.css"
import "./index.css"

import {loadBorders} from "./app/viewer/borderField.ts"
import {boardOf, FIRST_BOARD} from "./app/viewer/boards.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import SeasonMap, {BoardGeometry} from "./app/archive/SeasonMap.tsx"
import {FakeTurnBackend, SEASON_ZERO} from "./backends/fakeTurnBackend.ts"
import {TurnBackend} from "./backends/turn.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {languageInterceptor} from "./backends/language.ts"
import {speakChosenLanguage} from "./i18n/start.ts"

const speaking = speakChosenLanguage()

// A season's history can cross a change of board: each board is read once, as its first era is drawn.
const read = new Map<string, Promise<BoardGeometry>>()

function geometry(name: string): Promise<BoardGeometry> {
    let reading = read.get(name)
    if (reading === undefined) {
        const board = boardOf(name)
        reading = Promise.all([loadPointGeometryData(board), loadBorders(board.urls.borders)]).then(([{positions, cells}, {assignment, codes}]) => ({
            board, positions, cells, landmasses: {assignment, count: codes.length},
        }))
        read.set(name, reading)
    }
    return reading
}

function turnBackend(): TurnBackend {
    if (!(import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND)) {
        return new ConnectTurnBackend(newGamemasterServiceClient({
            baseUrl: API_BASE_URL,
            timeoutMs: 10_000,
            interceptors: [languageInterceptor(speaking.then((translator) => translator.language))],
        }))
    }
    const fake = new FakeTurnBackend(undefined, Date.now, [SEASON_ZERO])
    fake.replayOn(() => loadBorders(boardOf(FIRST_BOARD).urls.borders).then(({assignment, codes}) => new Map(
        [...assignment].flatMap((land, i) => land && codes[land] ? [[i + 1, codes[land]] as const] : []),
    )))
    return fake
}

const turn = turnBackend()
void speaking.then(() => createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <SeasonMap turn={turn} geometry={geometry}/>
    </StrictMode>,
))
