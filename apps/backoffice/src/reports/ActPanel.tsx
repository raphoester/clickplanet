import {ChatAdmin, PlanetAdmin} from "../clients.ts"
import {useOpenPlayer} from "../navigation.ts"
import {whoOf} from "../players/caller.ts"
import {InspectPanel} from "../players/InspectPanel.tsx"
import {BanAction, MuteAction} from "../players/Moderation.tsx"

type Props = {
    chat: ChatAdmin
    planet: PlanetAdmin
    accountId: string
    name: string
}

// What a report usually calls for. The player page has the rest: unban, revert, its address.
export function ActPanel({chat, planet, accountId, name}: Props) {
    const openPlayer = useOpenPlayer()
    const caller = {accountId}
    const who = whoOf(caller, name)

    return (
        <section className="panel" aria-label="Act on the reported account">
            <div className="panel-head">
                <h2>Act on {who}</h2>
                {openPlayer !== undefined && (
                    <button type="button" className="button-small" onClick={() => openPlayer({caller, name})}>
                        Every tool →
                    </button>
                )}
            </div>

            <MuteAction chat={chat} accountId={accountId} who={who}/>
            <BanAction planet={planet} caller={caller} who={who}/>

            <div className="act">
                <InspectPanel planet={planet} caller={caller}/>
            </div>
        </section>
    )
}
