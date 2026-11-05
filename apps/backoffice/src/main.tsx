import {StrictMode} from "react"
import {createRoot} from "react-dom/client"
import {App} from "./App.tsx"
import {requestOf, savePending} from "./connect/pending.ts"
import {callbackOf} from "./signIn/callback.ts"
import "./index.css"

const root = document.getElementById("root")
if (root === null) throw new Error("index.html has no #root")

// The code is good once: it leaves the address bar and the history before anything renders.
const callback = callbackOf(new URL(window.location.href))
if (callback !== undefined) window.history.replaceState(null, "", "/")

// An app's request waits in this tab while the operator signs in, and the page asks about it once it is in.
const request = requestOf(new URL(window.location.href))
if (request !== undefined) {
    savePending(request)
    window.history.replaceState(null, "", "/")
}

createRoot(root).render(
    <StrictMode>
        <App callback={callback}/>
    </StrictMode>,
)
