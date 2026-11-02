import {ReactNode, useState} from "react"
import {GAME_PATH, SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {AccountStore} from "./accountStore.ts"
import {rememberedSignIn, rememberedTrip} from "./rememberedSignIn.ts"
import SignInCallback from "./SignInCallback.tsx"
import {DiscordAlerts} from "../alerts/discordAlerts.ts"
import DiscordLinkCallback from "../alerts/DiscordLinkCallback.tsx"

export type SignInGateProps = {
    callback?: Callback
    account?: AccountStore
    alerts?: DiscordAlerts
    children: ReactNode
}

export default function SignInGate(props: SignInGateProps) {
    const [done, setDone] = useState(false)

    if (done || !props.callback) return props.children

    const onDone = () => {
        window.history.replaceState(null, "", GAME_PATH)
        setDone(true)
    }

    const alerts = props.alerts
    if (alerts && rememberedTrip() === "discordLink") {
        return <DiscordLinkCallback callback={props.callback}
                                    complete={alerts.complete}
                                    startAgain={alerts.leave}
                                    onDone={onDone}/>
    }

    if (!props.account) return props.children

    const account = props.account
    const remembered = rememberedSignIn()
    return <SignInCallback
        callback={props.callback}
        provider={remembered?.provider}
        complete={(code, state) => account.completeSignIn(code, state)}
        startAgain={remembered && (() => account.leaveFor(remembered.provider, remembered.intent))}
        onDone={onDone}/>
}
