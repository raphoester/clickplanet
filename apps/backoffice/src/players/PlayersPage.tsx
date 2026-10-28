import {useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {Opened} from "../navigation.ts"
import {idOf} from "./caller.ts"
import {PlayerList, Query} from "./PlayerList.tsx"
import {PlayerView} from "./PlayerView.tsx"

type Props = {
    opened: Opened | undefined
    onOpen: (opened: Opened | undefined) => void
    clients?: Clients
}

// The list is read again on the way back from a player, as the reports are: a ban shows on its row.
export function PlayersPage({opened, onOpen, clients = CLIENTS}: Props) {
    const [query, setQuery] = useState<Query>({kind: "top", limit: 20})

    if (opened !== undefined) {
        return (
            <PlayerView
                key={idOf(opened.caller)}
                clients={clients}
                opened={opened}
                onBack={() => onOpen(undefined)}
                onOpen={onOpen}
            />
        )
    }

    return <PlayerList planet={clients.planet} player={clients.player} query={query} onQuery={setQuery} onOpen={onOpen}/>
}
