import {createRoot} from "react-dom/client"
import {StrictMode} from "react"
import "./tokens.css"
import "./index.css"

import {loadBorders, loadProvinces} from "./app/viewer/borderField.ts"
import {replayGroundsOf} from "./backends/replayBackend.ts"
import {boardOf, FIRST_BOARD} from "./app/viewer/boards.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import SeasonMap, {BoardGeometry} from "./app/archive/SeasonMap.tsx"
import {FakeTurnBackend, SEASON_ZERO} from "./backends/fakeTurnBackend.ts"
import {TurnBackend} from "./backends/turn.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {languageInterceptor} from "./backends/language.ts"
import {speakChosenLanguage} from "./i18n/start.ts"
import {FlagLister} from "./backends/flags.ts"
import {ConnectFlagsBackend, newFlagServiceClient} from "./backends/flagsBackend.ts"
import {FakeFlagsBackend} from "./backends/fakeFlagsBackend.ts"
import {useFlags} from "./app/flags/flagsStore.ts"

const speaking = speakChosenLanguage()

// A season's history can cross a change of board: each board is read once, as its first era is drawn.
const read = new Map<string, Promise<BoardGeometry>>()

function geometry(name: string): Promise<BoardGeometry> {
    let reading = read.get(name)
    if (reading === undefined) {
        const board = boardOf(name)
        const borders = loadBorders(board.urls.borders)
        reading = Promise.all([loadPointGeometryData(board), borders, loadProvinces(board, borders)]).then(([{positions, cells}, read, provinces]) => ({
            board, positions, cells, grounds: replayGroundsOf(read, provinces),
        }))
        read.set(name, reading)
    }
    return reading
}

const faked = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND

const config = {
    baseUrl: API_BASE_URL,
    timeoutMs: 10_000,
    interceptors: [languageInterceptor(speaking.then((translator) => translator.language))],
}

function turnBackend(): TurnBackend {
    if (!faked) return new ConnectTurnBackend(newGamemasterServiceClient(config))
    const fake = new FakeTurnBackend(undefined, Date.now, [SEASON_ZERO])
    fake.replayOn(() => loadBorders(boardOf(FIRST_BOARD).urls.borders).then(({assignment, codes}) => new Map(
        [...assignment].flatMap((land, i) => land && codes[land] ? [[i + 1, codes[land]] as const] : []),
    )))
    return fake
}

// The flags opened since the build, named once: the archive has no stream to say the list changed.
function flagLister(): FlagLister {
    return faked ? new FakeFlagsBackend() : new ConnectFlagsBackend(newFlagServiceClient(config), newFlagServiceClient(config, true))
}

const turn = turnBackend()
useFlags.getState().serve(flagLister())
void speaking.then(() => createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <SeasonMap turn={turn} geometry={geometry}/>
    </StrictMode>,
))
