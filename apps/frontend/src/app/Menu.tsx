import {useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import Leaderboard from "./Leaderboard.tsx";
import MenuHeader from "./MenuHeader.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import Modal from "./components/Modal.tsx";
import DiscordButton from "./components/DiscordButton.tsx";
import BuyMeACoffee from "./components/BuyMeACoffee.tsx";
import {SwapIcon} from "./components/icons.tsx";
import {opensFolded} from "./compact.ts";
import "./Menu.css"

export type MenuProps = {
    country: Country,
    setCountry: (country: Country) => void,
    leaderboard: LeaderboardEntry[],
    tilesCount: number,
}

export default function Menu(props: MenuProps) {
    const [isOpen, setIsOpen] = useState(() => !opensFolded())
    const [pickingCountry, setPickingCountry] = useState(false)
    const [aboutOpen, setAboutOpen] = useState(false)
    const bodyId = useId()

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
                                <span className="menu-playing-name">
                                    <CountryFlag code={props.country.code}/>
                                    {props.country.name}
                                </span>
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
