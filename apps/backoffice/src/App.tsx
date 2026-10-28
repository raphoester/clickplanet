import {ReactNode, useCallback, useState} from "react"
import {authAdmin} from "./api.ts"
import {CLIENTS} from "./clients.ts"
import {FailureNotice} from "./components/FailureNotice.tsx"
import {Menu, Sidebar} from "./components/Menu.tsx"
import {HomePage} from "./home/HomePage.tsx"
import {PaintPanel} from "./map/PaintPanel.tsx"
import {ReassignPanel} from "./map/ReassignPanel.tsx"
import {ReplayPanel} from "./map/ReplayPanel.tsx"
import {ClipStatsPage} from "./clips/ClipStatsPage.tsx"
import {OpenPlayer, Opened, PageId, titleOf} from "./navigation.ts"
import {RulesPanel} from "./ops/RulesPanel.tsx"
import {AntibotRulesPanel} from "./ops/AntibotRulesPanel.tsx"
import {SchedulePanel} from "./ops/SchedulePanel.tsx"
import {SeasonsPanel} from "./ops/SeasonsPanel.tsx"
import {TimelinePanel} from "./ops/TimelinePanel.tsx"
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
import {useWide} from "./useWide.ts"

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
            return <Console operator={operator.value} onSignedOut={again}/>
    }
}

// The signed-in page: the menu picks one page, and its title says which.
function Console({operator, onSignedOut}: {operator: string, onSignedOut: () => void}) {
    const [page, setPage] = usePage()
    const wide = useWide()
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
                menu={wide ? undefined : <Menu page={page} onGo={go}/>}
                sidebar={wide ? <Sidebar page={page} onGo={go}/> : undefined}
                header={
                    <>
                        <code className="mono muted" title="Your account">{operator}</code>
                        <SignOutButton onSignedOut={onSignedOut}/>
                    </>
                }
            >
                <h2 className="page-title">{titleOf(page)}</h2>
                {page === "home" && <HomePage onGo={go}/>}
                {page === "notes" && <NoteList/>}
                {page === "reports" && <ReportsView/>}
                {page === "players" && <PlayersPage opened={player} onOpen={setPlayer}/>}
                {page === "reassign" && <ReassignPanel planet={CLIENTS.planet}/>}
                {page === "paint" && <PaintPanel planet={CLIENTS.planet}/>}
                {page === "replay" && <ReplayPanel planet={CLIENTS.planet}/>}
                {page === "clips" && <ClipStatsPage clips={CLIENTS.clips}/>}
                {page === "titles" && <TitlesPanel player={CLIENTS.player}/>}
                {page === "rules" && <RulesPanel planet={CLIENTS.planet}/>}
                {page === "antibotRules" && <AntibotRulesPanel antibot={CLIENTS.antibot}/>}
                {page === "schedule" && (
                    <SchedulePanel
                        gamemaster={CLIENTS.gamemaster} planet={CLIENTS.planet} antibot={CLIENTS.antibot}
                    />
                )}
                {page === "seasons" && <SeasonsPanel gamemaster={CLIENTS.gamemaster}/>}
                {page === "timeline" && <TimelinePanel gamemaster={CLIENTS.gamemaster} planet={CLIENTS.planet}/>}
            </Page>
        </OpenPlayer.Provider>
    )
}

type PageProps = {menu?: ReactNode, sidebar?: ReactNode, header: ReactNode, children: ReactNode}

function Page({menu, sidebar, header, children}: PageProps) {
    return (
        <div className="shell">
            {sidebar}
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
        </div>
    )
}
