import {useEffect, useState} from 'react'
import {ClickBudget, ClickBudgetSource} from "../../backends/clickBudget.ts"

/**
 * Subscribes to the server's readings and hands the latest one on, unchanged.
 *
 * It deliberately does **not** re-render as the bucket refills. This sits over
 * a WebGL scene with its own frame loop, and a React tree re-rendering ten times
 * a second beside it is a real cost for a number that moves by 0.1. The refill
 * is animated from the reading instead — see ClickBudgetMeter — so a render
 * here means the server said something new.
 *
 * `undefined` means no server reading has arrived: a backend with no throttle,
 * or one too old to report. Nothing is shown in that case rather than a
 * made-up allowance.
 */
export function useClickBudget(source?: ClickBudgetSource): ClickBudget | undefined {
    const [budget, setBudget] = useState<ClickBudget | undefined>()

    useEffect(() => {
        if (!source) return

        return source.watchClickBudget(setBudget)
    }, [source])

    return budget
}
