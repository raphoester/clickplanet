import {ReactNode, useState} from "react"
import {GAME_PATH, SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {AccountStore} from "./accountStore.ts"
import {rememberedSignIn} from "./rememberedSignIn.ts"
import SignInCallback from "./SignInCallback.tsx"

export type SignInGateProps = {
    callback?: Callback
    account?: AccountStore
    children: ReactNode
}

export default function SignInGate(props: SignInGateProps) {
    const [done, setDone] = useState(false)

    if (done || !props.callback || !props.account) return props.children

    const account = props.account
    const remembered = rememberedSignIn()
    return <SignInCallback
        callback={props.callback}
        provider={remembered?.provider}
        complete={(code, state) => account.completeSignIn(code, state)}
        startAgain={remembered && (() => account.leaveFor(remembered.provider, remembered.intent))}
        onDone={() => {
            window.history.replaceState(null, "", GAME_PATH)
            setDone(true)
        }}/>
}
