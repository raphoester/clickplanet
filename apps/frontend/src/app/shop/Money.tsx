import type {PlanPrice} from "../../domain/shop.ts"
import {pricePartsOf} from "./prices.ts"

// A price in the display face, its currency's sign in the text face: the display face's euro reads as a "<".
export default function Money({price, className = "shop-money"}: {price: PlanPrice, className?: string}) {
    return <span className={className}>
        {pricePartsOf(price).map(({kind, text}, i) => kind === "symbol"
            ? <span key={i} className="shop-currency">{text}</span>
            : text)}
    </span>
}
