// Stripe sends the buyer back to /play?shop=paid after a payment, and to /play?shop=cancelled after leaving its page.
export const SHOP_RETURN_PARAM = "shop"

export type ShopReturn = "paid" | "cancelled"

export function shopReturnOf(url: URL): ShopReturn | undefined {
    const back = url.searchParams.get(SHOP_RETURN_PARAM)
    return back === "paid" || back === "cancelled" ? back : undefined
}

export function withoutShopReturn(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(SHOP_RETURN_PARAM)
    return stripped
}
