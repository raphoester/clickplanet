import {savingLine} from "./prices.ts"
import {useShop} from "./shopStore.ts"
import "./Shop.css"

// The sale's cut on a door to the shop, while one runs: "-50%".
export default function SaleTag() {
    const sale = useShop((store) => store.sale)
    if (sale === undefined) return null
    return <span className="chip shop-sale-tag" aria-hidden="true">{savingLine(sale.percentOff)}</span>
}
