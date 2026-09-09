import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './index.css'

import {newClickServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import App from "./app/App.tsx"

const config = {
    baseUrl: import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol",
    timeoutMs: 2000,
}

const backend = new PlanetBackend(config, newClickServiceClient(config), 100)

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <App
            ownershipsGetter={backend}
            tileClicker={backend}
            updatesListener={backend}
        />
    </StrictMode>,
)
