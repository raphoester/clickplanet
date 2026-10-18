import {ReactNode, useCallback, useState} from "react"
import {authAdmin} from "./api.ts"
import {FailureNotice} from "./components/FailureNotice.tsx"
import {Callback} from "./signIn/callback.ts"
import {SignInCallback} from "./signIn/SignInCallback.tsx"
import {SignOutButton} from "./signIn/SignOutButton.tsx"
import {SupportPage} from "./support/SupportPage.tsx"
import {useCall} from "./useCall.ts"

export function App({callback}: {callback: Callback | undefined}) {
    const [returning, setReturning] = useState(callback)
    const leaveCallback = useCallback(() => setReturning(undefined), [])

    if (returning !== undefined) {
        return <Page header={null}><SignInCallback callback={returning} onDone={leaveCallback}/></Page>
    }
    return <Operator/>
}

const whoAmI = (signal: AbortSignal) => authAdmin.whoAmI({}, {signal}).then(answer => answer.accountId)

function Operator() {
    const [operator, again] = useCall(whoAmI)

    switch (operator.state) {
        case "loading":
            return <Page header={null}><p className="muted">Loading…</p></Page>
        case "failed":
            return <Page header={null}><FailureNotice failure={operator} onRetry={again}/></Page>
        case "done":
            return (
                <Page header={
                    <>
                        <code className="mono muted" title="Your account">{operator.value}</code>
                        <SignOutButton onSignedOut={again}/>
                    </>
                }>
                    <SupportPage/>
                </Page>
            )
    }
}

function Page({header, children}: {header: ReactNode, children: ReactNode}) {
    return (
        <div className="page">
            <header className="top">
                <h1>Clickplanet admin</h1>
                {header !== null && <div className="who">{header}</div>}
            </header>
            <main>{children}</main>
        </div>
    )
}
