import TitleBadge from "../titles/TitleBadge.tsx"
import {useBadge} from "./badgesStore.ts"

// The title an account wears, as a small medal beside its name: nothing until it is read, or when it wears none.
export default function WornTitle({accountId, size}: {accountId: string, size?: number}) {
    return <TitleBadge title={useBadge(accountId)?.wornTitle} size={size}/>
}
