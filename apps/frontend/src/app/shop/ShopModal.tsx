import {t} from "../../i18n/page.ts"
import Modal from "../components/Modal.tsx"
import {FlagIcon} from "../components/icons.tsx"
import {useFlagForm} from "../flags/flagFormStore.ts"
import {useFlagSubmissions} from "../flags/submissionsStore.ts"
import {useScreen} from "../hud/screenStore.ts"
import {planName, priceLine} from "./prices.ts"
import {useShop} from "./shopStore.ts"
import "./Shop.css"

// What the game sells, one card per product: today the region's flag, whose place takes the order.
export default function ShopModal() {
    const open = useScreen((screen) => screen.shopOpen)
    const setShopOpen = useScreen((screen) => screen.setShopOpen)
    const plans = useShop((store) => store.plans)
    const submitted = useFlagSubmissions((store) => store.submissions.length > 0)
    const showFlagPlace = useFlagForm((form) => form.show)

    if (!open) return null

    const close = () => setShopOpen(false)

    return <Modal title={t("menu.shop")} onClose={close}>
        <ul className="shop-products">
            <li className="panel-box shop-product">
                <div className="shop-product-head">
                    <span className="shop-product-icon"><FlagIcon size={26}/></span>
                    <h3 className="shop-product-name">{t("shop.flag.name")}</h3>
                </div>
                {plans.length > 0 && <ul className="shop-product-prices">
                    {plans.map((price) => <li key={price.plan} className="shop-product-price">
                        <span className="shop-plan-name">{planName(price.plan)}</span>
                        <span className="shop-plan-price">{priceLine(price)}</span>
                    </li>)}
                </ul>}
                <button type="button"
                        className="button button-action support-wide"
                        aria-haspopup="dialog"
                        onClick={() => {
                            close()
                            showFlagPlace()
                        }}>
                    {submitted ? t("shop.flag.yours") : t("shop.flag.add")}
                </button>
            </li>
        </ul>
    </Modal>
}
