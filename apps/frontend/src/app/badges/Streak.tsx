import StreakFlame from "../components/StreakFlame.tsx"
import {useBadge} from "./badgesStore.ts"

// An account's streak, as a flame beside its name: nothing until it is read, or while it is too short to show.
export default function Streak({accountId}: {accountId: string}) {
    return <StreakFlame days={useBadge(accountId)?.streak ?? 0}/>
}
