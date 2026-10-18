import {ReactNode, useCallback, useState} from "react"
import {authAdmin} from "./api.ts"
import {FailureNotice} from "./components/FailureNotice.tsx"
import {MapPage} from "./map/MapPage.tsx"
import {OpenPlayer, Opened, Section} from "./navigation.ts"
import {OpsPage} from "./ops/OpsPage.tsx"
import {PlayersPage} from "./players/PlayersPage.tsx"
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
                    <Sections/>
                </Page>
            )
    }
}

const SECTIONS: {section: Section, label: string}[] = [
    {section: "support", label: "Support"},
    {section: "players", label: "Players"},
    {section: "map", label: "Map"},
    {section: "ops", label: "Ops"},
]

function Sections() {
    const [section, setSection] = useState<Section>("support")
    const [player, setPlayer] = useState<Opened | undefined>(undefined)

    // A report opens its player on the Players page, where every tool on it is.
    const openPlayer = useCallback((opened: Opened) => {
        setPlayer(opened)
        setSection("players")
    }, [])

    // The tab of the section shown goes back to its top: Players to its list.
    const go = (to: Section) => {
        if (to === "players" && section === "players") setPlayer(undefined)
        setSection(to)
    }

    return (
        <OpenPlayer.Provider value={openPlayer}>
            <nav className="sections" aria-label="Sections">
                {SECTIONS.map(({section: each, label}) => (
                    <button key={each} type="button" aria-current={section === each ? "page" : undefined} onClick={() => go(each)}>
                        {label}
                    </button>
                ))}
            </nav>
            {section === "support" && <SupportPage/>}
            {section === "players" && <PlayersPage opened={player} onOpen={setPlayer}/>}
            {section === "map" && <MapPage/>}
            {section === "ops" && <OpsPage/>}
        </OpenPlayer.Provider>
    )
}

function Page({header, children}: {header: ReactNode, children: ReactNode}) {
    return (
        <div className="page">
            <header className="top">
                <h1>Clickplanet backoffice</h1>
                {header !== null && <div className="who">{header}</div>}
            </header>
            <main>{children}</main>
        </div>
    )
}
