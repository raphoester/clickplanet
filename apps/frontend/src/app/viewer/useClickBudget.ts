import {useEffect, useState} from 'react'
import {ClickBudget, ClickBudgetSource} from "../../backends/clickBudget.ts"

export function useClickBudget(source: ClickBudgetSource | undefined, countryId: string): ClickBudget | undefined {
    const [budget, setBudget] = useState<ClickBudget | undefined>()

    useEffect(() => {
        if (!source) return

        return source.watchClickBudget(setBudget)
    }, [source])

    useEffect(() => {
        source?.priceFor(countryId)
    }, [source, countryId])

    return budget
}
