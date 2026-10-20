import {useState} from "react"
import {authAdmin} from "../api.ts"

export function SignOutButton({onSignedOut}: {onSignedOut: () => void}) {
    const [going, setGoing] = useState(false)
    const [failed, setFailed] = useState(false)

    const signOut = () => {
        setGoing(true)
        setFailed(false)
        authAdmin.signOut({}).then(onSignedOut, () => {
            setGoing(false)
            setFailed(true)
        })
    }

    return (
        <button type="button" className="button-small" disabled={going} onClick={signOut}>
            {failed ? "Sign out failed, retry" : "Sign out"}
        </button>
    )
}
