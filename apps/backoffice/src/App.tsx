import {ReactNode, useCallback, useState} from "react"
import {authAdmin} from "./api.ts"
import {CLIENTS} from "./clients.ts"
import {FailureNotice} from "./components/FailureNotice.tsx"
import {ConnectPage} from "./connect/ConnectPage.tsx"
import {clearPending, pendingRequest} from "./connect/pending.ts"
import {Menu} from "./components/Menu.tsx"
import {PaintPanel} from "./map/PaintPanel.tsx"
import {ReassignPanel} from "./map/ReassignPanel.tsx"
import {ReplayPanel} from "./map/ReplayPanel.tsx"
import {OpenPlayer, Opened, PageId, titleOf} from "./navigation.ts"
import {TitlesPanel} from "./ops/TitlesPanel.tsx"
import {idOf} from "./players/caller.ts"
import {PlayersPage} from "./players/PlayersPage.tsx"
import {ReportsView} from "./reports/ReportsView.tsx"
import {Callback} from "./signIn/callback.ts"
import {SignInCallback} from "./signIn/SignInCallback.tsx"
import {SignOutButton} from "./signIn/SignOutButton.tsx"
import {NoteList} from "./support/NoteList.tsx"
import {useCall} from "./useCall.ts"
import {usePage} from "./usePage.ts"

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
    const [request, setRequest] = useState(pendingRequest)
    const dismiss = useCallback(() => {
        clearPending()
        setRequest(undefined)
    }, [])
    // Back to the app, whatever the operator said: nothing of the request is left for this tab.
    const leave = useCallback((url: string) => {
        clearPending()
        window.location.assign(url)
    }, [])

    switch (operator.state) {
        case "loading":
            return <Page header={null}><p className="muted">Loading…</p></Page>
        case "failed":
            return <Page header={null}><FailureNotice failure={operator} onRetry={again}/></Page>
        case "done":
            if (request !== undefined) {
                return (
                    <Page header={<code className="mono muted" title="Your account">{operator.value}</code>}>
                        <ConnectPage request={request} auth={authAdmin} leave={leave} onDismiss={dismiss}/>
                    </Page>
                )
            }
            return <Console operator={operator.value} onSignedOut={again}/>
    }
}

// The signed-in page: the burger menu picks one page, and its title says which.
function Console({operator, onSignedOut}: {operator: string, onSignedOut: () => void}) {
    const [page, setPage] = usePage()
    const [player, setPlayer] = useState<Opened | undefined>(undefined)

    // A report opens its player on the Players page, where every tool on it is. The player already open keeps its row.
    const openPlayer = useCallback((opened: Opened) => {
        setPlayer(shown => shown !== undefined && idOf(shown.caller) === idOf(opened.caller) ? shown : opened)
        setPage("players")
    }, [setPage])

    // Picking the page shown goes back to its top: Players to its list.
    const go = (to: PageId) => {
        if (to === "players" && page === "players") setPlayer(undefined)
        setPage(to)
    }

    return (
        <OpenPlayer.Provider value={openPlayer}>
            <Page
                menu={<Menu page={page} onGo={go}/>}
                header={
                    <>
                        <code className="mono muted" title="Your account">{operator}</code>
                        <SignOutButton onSignedOut={onSignedOut}/>
                    </>
                }
            >
                <h2 className="page-title">{titleOf(page)}</h2>
                {page === "notes" && <NoteList/>}
                {page === "reports" && <ReportsView/>}
                {page === "players" && <PlayersPage opened={player} onOpen={setPlayer}/>}
                {page === "reassign" && <ReassignPanel planet={CLIENTS.planet}/>}
                {page === "paint" && <PaintPanel planet={CLIENTS.planet}/>}
                {page === "replay" && <ReplayPanel planet={CLIENTS.planet}/>}
                {page === "titles" && <TitlesPanel player={CLIENTS.player}/>}
            </Page>
        </OpenPlayer.Provider>
    )
}

type PageProps = {menu?: ReactNode, header: ReactNode, children: ReactNode}

function Page({menu, header, children}: PageProps) {
    return (
        <div className="page">
            <header className="top">
                <div className="brand">
                    {menu}
                    <h1>Clickplanet backoffice</h1>
                </div>
                {header !== null && <div className="who">{header}</div>}
            </header>
            <main>{children}</main>
        </div>
    )
}
