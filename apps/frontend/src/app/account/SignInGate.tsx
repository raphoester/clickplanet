import {ReactNode, useState} from "react"
import {SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {AccountStore} from "./accountStore.ts"
import {rememberedProvider} from "./rememberedProvider.ts"
import SignInCallback from "./SignInCallback.tsx"

export type SignInGateProps = {
    /** What the provider sent, when this page load is the callback. */
    callback?: Callback
    /** Absent in fake mode: a callback there goes straight to the game. */
    account?: AccountStore
    children: ReactNode
}

/**
 * The callback page while a sign-in finishes, then the game — in place, with
 * no reload. The game then starts with the click token the sign-in invalidated,
 * so its first click mints one that carries the new account.
 */
export default function SignInGate(props: SignInGateProps) {
    const [done, setDone] = useState(false)

    if (done || !props.callback || !props.account) return props.children

    const account = props.account
    const provider = rememberedProvider()
    return <SignInCallback
        callback={props.callback}
        complete={(code, state) => account.completeSignIn(code, state)}
        startAgain={provider && (() => account.leaveFor(provider))}
        onDone={() => {
            window.history.replaceState(null, "", "/")
            setDone(true)
        }}/>
}
