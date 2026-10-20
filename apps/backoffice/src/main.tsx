import {StrictMode} from "react"
import {createRoot} from "react-dom/client"
import {App} from "./App.tsx"
import {callbackOf} from "./signIn/callback.ts"
import "./index.css"

const root = document.getElementById("root")
if (root === null) throw new Error("index.html has no #root")

// The code is good once: it leaves the address bar and the history before anything renders.
const callback = callbackOf(new URL(window.location.href))
if (callback !== undefined) window.history.replaceState(null, "", "/")

createRoot(root).render(
    <StrictMode>
        <App callback={callback}/>
    </StrictMode>,
)
