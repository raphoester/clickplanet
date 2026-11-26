import {Sale} from "../../domain/shop.ts"
import {rich} from "../../i18n/rich.tsx"
import {useNow} from "../season/useNow.ts"
import {saleLeftOf} from "./prices.ts"
import "./Shop.css"

// The sale's name and the time it has left, to the second: "Black Friday · ends in 2d 04h 12m 09s".
export default function SaleClock({sale}: {sale: Sale}) {
    const now = useNow()
    return <p className="shop-sale-clock" role="timer">
        {rich("shop.onSale.endsIn", {time: <span className="shop-sale-left">{saleLeftOf(sale, now)}</span>})}
    </p>
}
