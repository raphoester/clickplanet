import './BuyMeACoffee.css'

export type BuyMeACoffeeProps = {
    onOpen?: () => void
}

export default function BuyMeACoffee({onOpen}: BuyMeACoffeeProps) {
    return <a
        href="https://buymeacoffee.com/raphoester"
        target="_blank"
        rel="noopener noreferrer"
        className="button button-gold button-coffee"
        onClick={onOpen}
    >
        Buy me a coffee
    </a>
};
