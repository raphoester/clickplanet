import {ReactNode} from "react"
import {authAdmin} from "./api.ts"
import {FailureNotice} from "./components/FailureNotice.tsx"
import {SupportPage} from "./support/SupportPage.tsx"
import {useCall} from "./useCall.ts"

const whoAmI = (signal: AbortSignal) => authAdmin.getOperator({}, {signal}).then(answer => answer.accountId)

export function App() {
    const [operator, again] = useCall(whoAmI)

    switch (operator.state) {
        case "loading":
            return <Page operator=""><p className="muted">Loading…</p></Page>
        case "failed":
            return <Page operator=""><FailureNotice failure={operator} onRetry={again}/></Page>
        case "done":
            return <Page operator={operator.value}><SupportPage/></Page>
    }
}

function Page({operator, children}: {operator: string, children: ReactNode}) {
    return (
        <div className="page">
            <header className="top">
                <h1>Clickplanet admin</h1>
                {operator !== "" && <code className="mono muted" title="Your account">{operator}</code>}
            </header>
            <main>{children}</main>
        </div>
    )
}
