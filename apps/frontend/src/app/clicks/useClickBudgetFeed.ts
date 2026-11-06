import {useEffect} from 'react'
import {ClickBudgetSource} from "../../backends/clickBudget.ts"
import {useClickBudget} from "./clickBudgetStore.ts"

export function useClickBudgetFeed(source: ClickBudgetSource, countryId: string): void {
    useEffect(() => {
        const stop = source.watchClickBudget(useClickBudget.getState().hear)
        return () => {
            stop()
            useClickBudget.setState({budget: undefined})
        }
    }, [source])

    useEffect(() => {
        source.priceFor(countryId)
    }, [source, countryId])
}
