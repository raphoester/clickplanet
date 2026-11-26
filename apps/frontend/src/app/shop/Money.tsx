import type {PlanPrice} from "../../domain/shop.ts"
import {pricePartsOf} from "./prices.ts"
import "./Shop.css"

export type MoneyProps = {
    price: PlanPrice
    className?: string
    // A price no longer asked: the line crosses its digits alone, as one through the euro's bars reads as a "£".
    struck?: boolean
}

// A price in the display face, its currency's sign in the text face: the display face's euro reads as a "<".
export default function Money({price, className = "shop-money", struck = false}: MoneyProps) {
    return <span className={className}>
        {pricePartsOf(price).map(({kind, text}, i) => {
            if (kind === "symbol") return <span key={i} className="shop-currency">{text}</span>
            return struck ? <s key={i} className="shop-struck">{text}</s> : text
        })}
    </span>
}
