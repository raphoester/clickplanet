import {FormEvent, useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {Opened} from "../navigation.ts"
import {callerOf, idOf} from "./caller.ts"
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

    return (
        <>
            <OpenForm onOpen={onOpen}/>
            <PlayerList planet={clients.planet} query={query} onQuery={setQuery} onOpen={onOpen}/>
        </>
    )
}

function OpenForm({onOpen}: {onOpen: (opened: Opened) => void}) {
    const [typed, setTyped] = useState("")
    const caller = callerOf(typed)

    const open = (event: FormEvent) => {
        event.preventDefault()
        if (caller !== undefined) onOpen({caller})
    }

    return (
        <form className="open-form" aria-label="Open a player" onSubmit={open}>
            <label className="field">
                <span className="muted">Account id or address</span>
                <input
                    value={typed}
                    placeholder="0b7e5b6c-… or 203.0.113.7"
                    autoCapitalize="none"
                    autoComplete="off"
                    spellCheck={false}
                    onChange={event => setTyped(event.target.value)}
                />
            </label>
            <button type="submit" className="button" disabled={caller === undefined}>Open</button>
        </form>
    )
}
