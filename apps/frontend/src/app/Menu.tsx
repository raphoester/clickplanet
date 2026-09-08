import {useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import Leaderboard from "./Leaderboard.tsx";
import MenuHeader from "./MenuHeader.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import Modal from "./components/Modal.tsx";
import DiscordButton from "./components/DiscordButton.tsx";
import BuyMeACoffee from "./components/BuyMeACoffee.tsx";
import {SwapIcon} from "./components/icons.tsx";
import "./Menu.css"

export type MenuProps = {
    country: Country,
    setCountry: (country: Country) => void,
    leaderboard: LeaderboardEntry[],
    tilesCount: number,
}

/** Where the stylesheets switch the card from a floating panel to a full-width sheet. */
const COMPACT = "(max-width: 768px)"

/**
 * Whether the card should open folded. Read once, as a starting position rather
 * than a binding: a player who opens the card should not have it shut again
 * because they rotated the phone.
 *
 * jsdom has no matchMedia, and neither do the tests that render this.
 */
const opensFolded = () => window.matchMedia?.(COMPACT).matches ?? false

/**
 * The card over the globe: who you play for, how everyone is doing, and the way
 * to change either.
 *
 * Three navigation gestures, each used for exactly one thing, which is the only
 * reason they cannot be confused for one another:
 *
 *  - the chevron in the header folds the whole card, and nothing else collapses;
 *  - the back arrow exists only inside the country picker, the one place you can
 *    drill into;
 *  - the × belongs to About, which opens over everything instead of taking the
 *    card's content slot.
 *
 * That last one is why About is a Modal and not a MenuPanel. It is a detour
 * rather than a step, and giving it the card's slot meant a "Back" button that
 * contradicted the "Close" button one level up.
 */
export default function Menu(props: MenuProps) {
    const [isOpen, setIsOpen] = useState(() => !opensFolded())
    const [pickingCountry, setPickingCountry] = useState(false)
    const [aboutOpen, setAboutOpen] = useState(false)
    const bodyId = useId()

    /**
     * Drilling in unmounts the button that was clicked, so there is no element
     * left to hand focus back to when the panel closes. Modal restores its own
     * opener; only the panel needs this.
     */
    const cameFromChange = useRef(false)
    const changeButton = useRef<HTMLButtonElement>(null)

    const openPicker = () => {
        cameFromChange.current = true
        setPickingCountry(true)
    }

    useEffect(() => {
        if (pickingCountry || !cameFromChange.current) return
        cameFromChange.current = false
        changeButton.current?.focus()
    }, [pickingCountry])

    const pickCountry = (country: Country) => {
        props.setCountry(country)
        setPickingCountry(false)
    }

    return <>
        <div className="menu">
            <MenuHeader country={props.country}
                        rank={rankOf(props.leaderboard, props.country)}
                        isOpen={isOpen}
                        onToggle={() => setIsOpen(!isOpen)}
                        bodyId={bodyId}/>

            {isOpen && <div className="menu-body" id={bodyId}>
                {pickingCountry
                    ? <MenuPanel title="Change country" onClose={() => setPickingCountry(false)}>
                        <CountryPicker country={props.country} setCountry={pickCountry}/>
                    </MenuPanel>
                    : <>
                        <div className="menu-playing">
                            <span className="menu-label">You’re playing for</span>
                            <div className="menu-playing-row">
                                <span className="menu-playing-name">{props.country.name}</span>
                                <button ref={changeButton}
                                        type="button"
                                        className="button button-mini menu-change"
                                        onClick={openPicker}>
                                    <SwapIcon/>
                                    <span>Change</span>
                                </button>
                            </div>
                        </div>

                        <Leaderboard data={props.leaderboard}
                                     tilesCount={props.tilesCount}
                                     highlight={props.country}/>

                        <div className="menu-actions">
                            <button type="button"
                                    className="button button-ghost"
                                    onClick={() => setAboutOpen(true)}>
                                About
                            </button>
                            <DiscordButton message="Discord"/>
                        </div>
                    </>}
            </div>}
        </div>

        {aboutOpen && <Modal title="About ClickPlanet"
                             footer={<BuyMeACoffee/>}
                             onClose={() => setAboutOpen(false)}>
            <About/>
        </Modal>}
    </>
}
