import {useEffect, useRef, useState} from 'react';
import {
    AllegianceSource,
    BankFullError,
    Bomber,
    BonusListener,
    BonusLostError,
    ChargesRefresher,
    OwnershipsGetter,
    QuizMaster,
    Refiller,
    Shielder,
    TileClicker,
    TileHolders,
    UpdatesListener,
} from "../../backends/backend.ts";
import BombNews from "../components/BombNews.tsx";
import FortifyNews from "../components/FortifyNews.tsx";
import FortifyCard from "../components/FortifyCard.tsx";
import FortifyHint from "../components/FortifyHint.tsx";
import {useFortifyGuide} from "../components/useFortifyGuide.ts";
import {Fortified} from "../../domain/fortify.ts";
import ConnectionLost from "../components/ConnectionLost.tsx";
import {ConnectionSource} from "../../backends/connection.ts";
import {useConnection} from "./useConnection.ts";
import Quiz from "../quiz/Quiz.tsx";
import {useQuiz} from "../quiz/useQuiz.ts";
import {ChatBackend} from "../../backends/chat.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu, {BoardSheet, MenuTab, YouPlace} from "../Menu.tsx";
import {youLabel} from "../youLabel.ts";
import BonusAward from "../components/BonusAward.tsx";
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx";
import ClicksPanel from "../components/ClicksPanel.tsx";
import Inventory from "../components/Inventory.tsx";
import {useBonusGuide} from "../components/useBonusGuide.ts";
import {useBoardGuide} from "../useBoardGuide.ts";
import {isFirstWin} from "../../domain/bonusGuide.ts";
import SessionUnavailableModal from "../components/SessionUnavailableModal.tsx";
import VPNBlockedModal from "../components/VPNBlockedModal.tsx";
import BannedModal from "../components/BannedModal.tsx";
import SharePlace from "../share/SharePlace.tsx";
import SharePreview from "../share/SharePreview.tsx";
import {useSharePicture} from "../share/useSharePicture.ts";
import {shareStats} from "../../domain/shareCard.ts";
import {Countries, Country} from "../../domain/countries.ts";
import {useAllegiance} from "./useAllegiance.ts";
import {heldLine} from "../../domain/allegiance.ts";
import {ClickBudgetSource, now as budgetNow, tokensAt} from "../../backends/clickBudget.ts";
import {useClickBudget} from './useClickBudget.ts';
import {useCountryStorage} from './useCountryStorage.ts';
import {GlobeStatus, useGlobe} from './useGlobe.ts';
import {useSound} from '../sound/useSound.ts';
import {useDisplaySettings} from './useDisplaySettings.ts';
import SettingsPlace from "../settings/SettingsPlace.tsx";
import AnthemControls from "../anthem/AnthemControls.tsx";
import {useAnthem} from "../anthem/useAnthem.ts";
import {AccountStore} from "../account/accountStore.ts";
import {useAccount} from "../account/useAccount.ts";
import {hueOf} from "../../domain/authorColor.ts";
import {PlayerInfoBackend, PlayerLine, PlayerTitle, PresenceBackend} from "../../backends/player.ts";
import PlayerCard from "../players/PlayerCard.tsx";
import HolderCard from "../players/HolderCard.tsx";
import TerritoryCard from "../territory/TerritoryCard.tsx";
import {useTerritoryCard} from "../territory/useTerritoryCard.ts";
import {useTileHolder} from "../players/useTileHolder.ts";
import TitleUnlocked from "../titles/TitleUnlocked.tsx";
import {usePresence} from "../players/usePresence.ts";
import {useRoster} from "../players/useRoster.ts";
import SignInPitchModal from "../account/SignInPitchModal.tsx";
import {LeaderboardEntry, rankOf} from "../../domain/leaderboard.ts";
import {SeasonBackend} from "../../backends/season.ts";
import {useSeason} from "../season/useSeason.ts";
import SeasonChip, {ChampionChip, SeasonDetails} from "../season/SeasonChip.tsx";
import {HallOfFameBackend} from "../../backends/hallOfFame.ts";
import {useHallOfFame} from "../hallOfFame/useHallOfFame.ts";
import HallOfFame from "../hallOfFame/HallOfFame.tsx";
import HallOfFameButton from "../hallOfFame/HallOfFameButton.tsx";
import {useCompact} from "../compact.ts";
import Sheet from "../hud/Sheet.tsx";
import StatusBar from "../hud/StatusBar.tsx";
import TabBar from "../hud/TabBar.tsx";
import MissionsPlace, {MissionsPlaceProps} from "../missions/MissionsPlace.tsx";
import {useMissions} from "../missions/useMissions.ts";
import {MissionsBackend} from "../../backends/missions.ts";
import {awardOf, waiting} from "../../domain/missions.ts";
import {usePaidMissions} from "../missions/usePaidMissions.ts";
import {StandingsBackend} from "../../backends/standings.ts";
import {SupportBackend} from "../../backends/support.ts";
import {pageContextOf} from "../support/pageContext.ts";
import {BoardStandings, BoardView} from "../standings/BoardViews.tsx";
import {useRace} from "../standings/useRace.ts";
import RoundReveal from "../season/RoundReveal.tsx";
import {useRoundReveal} from "../season/useRoundReveal.ts";
import {revealKeyOf} from "../../domain/roundReveal.ts";
import {reigningChampionOf} from "../../domain/champion.ts";
import {countryLines, CountryOrder, shownOrder} from "../../domain/race.ts";
import {acceptedClicks} from "./acceptedClicks.ts";
import "./Viewer.css"

const NO_LEADERBOARD: readonly LeaderboardEntry[] = []

// After the wave has run over the land.
const FORTIFY_CARD_DELAY_MS = 1600

type SheetName = "board" | "missions" | "chat" | "you" | "share" | "settings" | "season" | "clicks"

export type ViewerProps = {
    sharedCountry?: Country
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudgetSource?: ClickBudgetSource
    connection?: ConnectionSource
    bonusListener?: BonusListener
    quizMaster?: QuizMaster
    bomber?: Bomber
    refiller?: Refiller
    shielder?: Shielder
    allegiance?: AllegianceSource
    chatBackend?: ChatBackend
    account?: AccountStore
    presence?: PresenceBackend
    playerInfo?: PlayerInfoBackend
    tileHolders?: TileHolders
    season?: SeasonBackend
    standings?: StandingsBackend
    support?: SupportBackend
    hallOfFame?: HallOfFameBackend
    missions?: MissionsBackend
    chargesRefresher?: ChargesRefresher
}

// Planet grants a mission's reward from a command, after the read that says it is paid.
const GRANT_LANDS_MS = 2_000

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage(props.sharedCountry)
    const allegiance = useAllegiance(props.allegiance)
    useEffect(() => {
        const held = allegiance && allegiance.countryId !== countryState.code ? Countries.get(allegiance.countryId) : undefined
        if (held) handleSetCountry(held)
    }, [allegiance, countryState.code, handleSetCountry])
    const pickCountry = async (country: Country) => {
        if (country.code !== countryState.code) await props.allegiance?.pledgeAllegiance(country.code)
        handleSetCountry(country)
    }
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const connection = useConnection(props.connection)
    const sound = useSound()
    const display = useDisplaySettings()
    const account = useAccount(props.account)
    const username = account.kind === 'ready' ? account.username : undefined
    const ownAccountId = account.kind === 'ready' ? account.accountId : undefined
    const color = account.kind === 'ready' ? account.color : undefined

    usePresence(props.presence, {countryCode: countryState.code, username, color})

    const quiz = useQuiz(props.quizMaster, countryState.code, sound.play)
    const [unlocked, setUnlocked] = useState<readonly PlayerTitle[]>([])
    const roster = useRoster(props.presence, (title) => setUnlocked((queue) => [...queue, title]))
    const season = useSeason(props.season)
    const race = useRace(props.standings)
    const reveal = useRoundReveal(race)
    const champion = reigningChampionOf(season, race?.closed)
    const hall = useHallOfFame(props.hallOfFame, race?.closed?.finale ? race.closed.season : undefined)
    const [hallOpen, setHallOpen] = useState(false)
    const [countryOrder, setCountryOrder] = useState<CountryOrder>("season")
    const boardGuide = useBoardGuide()
    const [pitchOpen, setPitchOpen] = useState(false)
    const [openPlayer, setOpenPlayer] = useState<PlayerLine>()
    const onOpenPlayer = props.playerInfo ? setOpenPlayer : undefined
    const guest = account.kind === 'ready' && account.offered.length > 0 && account.me.linked.length === 0

    const compact = useCompact()
    const [sheet, setSheet] = useState<SheetName>()
    const [menuTab, setMenuTab] = useState<MenuTab>("board")
    const [seasonOpen, setSeasonOpen] = useState(false)
    const [clicksOpen, setClicksOpen] = useState(false)
    const [unread, setUnread] = useState(0)
    const [boardView, setBoardView] = useState<BoardView>("countries")
    const [accepted] = useState(acceptedClicks)
    const toggleSheet = (name: SheetName) => setSheet((current) => current === name ? undefined : name)
    const closeSheet = () => setSheet(undefined)

    useEffect(() => {
        setSheet(undefined)
        setSeasonOpen(false)
        setClicksOpen(false)
    }, [compact])

    const {
        status,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        graphics,
        refusals,
        vpnBlocked,
        dismissVPNBlocked,
        bannedUntil,
        dismissBanned,
        sessionUnavailable,
        dismissSessionUnavailable,
        lockedOut,
        award,
        showAward,
        dismissAward,
        charges,
        rules,
        bombArmed,
        toggleBomb,
        switches,
        toggleSwitch,
        notice,
        lastBomb,
        dismissBomb,
        lastFortified,
        dismissFortified,
        lastClose,
        dismissClose,
        pointing,
        pointingTerritory,
        territoryOf,
        showTerritory,
    } = useGlobe({
        container,
        tileClicker: props.tileClicker,
        ownershipsGetter: props.ownershipsGetter,
        updatesListener: props.updatesListener,
        clickBudget: props.clickBudgetSource,
        bonusListener: props.bonusListener,
        bomber: props.bomber,
        shielder: props.shielder,
        playSound: sound.play,
        onClickAccepted: accepted.record,
        country: countryState,
        clickHue: hueOf(color),
        mapView: display.settings.mapView,
        rendering: display.settings.rendering,
    })

    const tileHolder = useTileHolder(props.tileHolders, props.playerInfo, pointing, ownAccountId)
    const territoryCard = useTerritoryCard(pointingTerritory, territoryOf, showTerritory)

    const bonusGuide = useBonusGuide()

    const fortifyGuide = useFortifyGuide()
    const [fortifyCard, setFortifyCard] = useState<Fortified | undefined>()
    useEffect(() => {
        if (!lastFortified || fortifyGuide.seen) return
        const timer = setTimeout(() => setFortifyCard(lastFortified.fortified), FORTIFY_CARD_DELAY_MS)
        return () => clearTimeout(timer)
    }, [lastFortified, fortifyGuide.seen])
    // Moments show one at a time: the newest of them.
    const newest = [lastBomb, lastFortified, lastClose].reduce<{at: number} | undefined>(
        (latest, moment) => moment && (!latest || moment.at > latest.at) ? moment : latest, undefined)

    const refiller = props.refiller
    const spendRefill = refiller && (() => {
        if (clickBudget && tokensAt(clickBudget, budgetNow()) >= clickBudget.capacity) return false
        refiller.useRefill(countryState.code).catch((e) => {
            if (e instanceof BankFullError || e instanceof BonusLostError) return
            console.error("could not use the refill", e)
        })
        return true
    })

    const order = shownOrder(race, countryOrder)
    const lines = countryLines(leaderboard, race, order)
    const anthem = useAnthem(status.state === 'ready' ? lines : NO_LEADERBOARD, order, sound.settings)

    const stats = shareStats(lines, countryState)
    const {shot, taking, take, discard} = useSharePicture(capture, stats)

    const openPitch = guest ? () => {
        closeSheet()
        setClicksOpen(false)
        setPitchOpen(true)
    } : undefined

    const linked = account.kind === 'ready' && account.me.linked.length > 0
    const toll = rules?.toll ?? []
    const held = leaderboard.find((entry) => entry.country.code === countryState.code)?.tiles ?? 0
    const standings: BoardStandings | undefined = props.standings && {
        backend: props.standings,
        caller: {accountId: ownAccountId, username, color, linked},
        listenForClicks: accepted.listenForClicks,
        onSignIn: clickBudget?.linkedMultiplier ? openPitch : undefined,
        onOpenPlayer,
        view: boardView,
        onView: setBoardView,
    }

    const board = {
        country: countryState,
        setCountry: pickCountry,
        allegiance,
        leaderboard,
        tileDeltas,
        tilesCount,
        anthem: <AnthemControls anthem={anthem} settings={sound.settings} onChange={sound.setSettings}/>,
        standings,
        race,
        countryOrder,
        onCountryOrder: setCountryOrder,
        guided: boardGuide.guided,
        onGuided: boardGuide.markGuided,
    }
    const missionsState = useMissions(props.missions, allegiance?.countryId, accepted.listenForClicks)
    usePaidMissions(missionsState, (paid) => {
        if (rules) paid.forEach((mission) => showAward(awardOf(mission.reward, rules)))
        const refresh = () => props.chargesRefresher?.refreshCharges().catch((e) => console.error("could not read the charges held", e))
        refresh()
        setTimeout(refresh, GRANT_LANDS_MS)
    })
    const missions: MissionsPlaceProps | undefined = props.missions && {
        state: missionsState,
        onSignIn: guest ? openPitch ?? (() => setSheet("you")) : undefined,
    }
    const missionsReady = missionsState.kind === "ready" ? waiting(missionsState.day.missions) : 0

    const you = {
        account: props.account,
        linkedMultiplier: clickBudget?.linkedMultiplier,
        playerInfo: props.playerInfo,
        listenForClicks: accepted.listenForClicks,
    }
    const share = {
        onTakePicture: () => {
            closeSheet()
            take()
        },
        taking,
    }
    const settings = {
        display: {settings: display.settings, onChange: display.setSettings},
        sound: {settings: sound.settings, onChange: sound.setSettings, preview: sound.preview},
        support: props.support,
        pageContext: () => pageContextOf(window, graphics(), connection === "down"),
    }
    const clicks = <ClicksPanel budget={clickBudget}
                                countryName={countryState.name}
                                share={tilesCount > 0 ? held / tilesCount : undefined}
                                toll={toll}
                                onSignIn={openPitch}/>

    const ready = status.state === 'ready'
    const fame = (compact: boolean) => hall && hall.length > 0 && <HallOfFameButton compact={compact}
                                                                                    besideSeason={champion !== undefined || season !== undefined}
                                                                                    onOpen={() => setHallOpen(true)}/>

    return <>
        <div ref={container} className="viewer-canvas"/>

        {!ready && <StatusCard status={status}/>}

        {ready && !compact && <Menu {...board} {...you} {...share} {...settings}
                                    missions={missions}
                                    tab={menuTab}
                                    onTab={setMenuTab}/>}

        {ready && !compact && fame(false)}

        {ready && !compact && champion && <ChampionChip champion={champion} compact={false}/>}
        {ready && !compact && !champion && season && <SeasonChip season={season}
                                                                 compact={false}
                                                                 open={seasonOpen}
                                                                 onToggle={() => setSeasonOpen((open) => !open)}/>}

        {ready && compact && <StatusBar country={countryState}
                                        rank={rankOf(lines, countryState)}
                                        boardOpen={sheet === "board"}
                                        onOpenBoard={() => toggleSheet("board")}
                                        trailing={<>
                                            {fame(true)}
                                            {champion
                                                ? <ChampionChip champion={champion} compact/>
                                                : season && <SeasonChip season={season}
                                                                        compact
                                                                        open={sheet === "season"}
                                                                        onToggle={() => toggleSheet("season")}/>}
                                        </>}/>}

        {ready && <ClickBudgetMeter budget={clickBudget}
                                    refusals={refusals}
                                    said={lockedOut && {text: heldLine(lockedOut.allegiance, Date.now()), id: lockedOut.id}}
                                    onSignIn={openPitch}
                                    compact={compact}
                                    open={compact ? sheet === "clicks" : clicksOpen}
                                    onToggleOpen={compact ? () => toggleSheet("clicks") : () => setClicksOpen((open) => !open)}
                                    popover={clicks}>
            {props.bonusListener && <Inventory charges={charges}
                                               rules={rules}
                                               switches={switches}
                                               onToggle={toggleSwitch}
                                               bombArmed={bombArmed}
                                               onToggleBomb={props.bomber ? toggleBomb : undefined}
                                               onUseRefill={spendRefill}
                                               onToggleShield={props.shielder ? () => toggleSwitch("shield") : undefined}
                                               notice={notice}
                                               guide={bonusGuide.guide}
                                               onUsed={bonusGuide.markUsed}/>}
        </ClickBudgetMeter>}

        {ready && <ChatPanel
            backend={props.chatBackend}
            country={countryState}
            playSound={sound.play}
            accountId={ownAccountId}
            username={username}
            onOpenPlayer={onOpenPlayer}
            reporter={props.support}
            players={roster.kind === 'ready' ? roster.entries : undefined}
            compact={compact}
            open={compact ? sheet === "chat" : undefined}
            onOpenChange={compact ? (open) => setSheet(open ? "chat" : undefined) : undefined}
            onUnread={setUnread}
        />}

        {ready && compact && <>
            {sheet === "board" && <BoardSheet {...board} onClose={closeSheet}/>}
            {sheet === "missions" && missions && <Sheet title="Missions" onClose={closeSheet}>
                <MissionsPlace {...missions}/>
            </Sheet>}
            {sheet === "you" && account.kind === 'ready' && <Sheet title={youLabel(linked)} onClose={closeSheet}>
                <YouPlace {...you}/>
            </Sheet>}
            {sheet === "share" && <Sheet title="Share" onClose={closeSheet}>
                <SharePlace stats={stats} {...share}/>
            </Sheet>}
            {sheet === "settings" && <Sheet title="Settings" onClose={closeSheet}>
                <SettingsPlace {...settings}/>
            </Sheet>}
            {sheet === "season" && season && <Sheet title={`Season ${season.number}`} onClose={closeSheet}>
                <SeasonDetails season={season}/>
            </Sheet>}
            {sheet === "clicks" && <Sheet title="Your clicks" onClose={closeSheet}>{clicks}</Sheet>}

            <TabBar open={sheet}
                    onOpen={toggleSheet}
                    chat={props.chatBackend !== undefined}
                    unread={unread}
                    you={account.kind === 'ready' ? (linked ? "player" : "guest") : undefined}
                    missions={missions ? missionsReady : undefined}/>
        </>}

        {shot && <SharePreview shot={shot}
                               stats={stats}
                               onClose={discard}/>}

        {pitchOpen && guest && props.account && clickBudget?.linkedMultiplier && <SignInPitchModal
            state={account}
            store={props.account}
            multiplier={clickBudget.linkedMultiplier}
            onClose={() => setPitchOpen(false)}/>}

        {hallOpen && hall && <HallOfFame hall={hall}
                                         countryCode={countryState.code}
                                         accountId={ownAccountId}
                                         onOpenPlayer={onOpenPlayer && ((player) => {
                                             setHallOpen(false)
                                             onOpenPlayer(player)
                                         })}
                                         onClose={() => setHallOpen(false)}/>}

        {tileHolder.holder && <HolderCard key={`holder-${tileHolder.holder.id}`}
                                                              holder={tileHolder.holder}
                                                              onHold={tileHolder.hold}
                                                              onOpenPlayer={onOpenPlayer && ((player) => {
                                                                  tileHolder.dismiss()
                                                                  onOpenPlayer(player)
                                                              })}
                                                              onClose={tileHolder.dismiss}/>}

        {territoryCard.shown && <TerritoryCard key={`territory-${territoryCard.shown.id}`}
                                               shown={territoryCard.shown}
                                               flag={countryState.code}
                                               onHold={territoryCard.hold}
                                               onClose={territoryCard.dismiss}/>}

        {openPlayer && props.playerInfo && <PlayerCard key={openPlayer.accountId}
                                                       player={openPlayer}
                                                       backend={props.playerInfo}
                                                       reporter={openPlayer.accountId === ownAccountId ? undefined : props.support}
                                                       onClose={() => setOpenPlayer(undefined)}/>}

        {unlocked.length > 0 && <TitleUnlocked key={unlocked[0].id}
                                               title={unlocked[0]}
                                               play={sound.play}
                                               onWear={props.account?.wearTitle}
                                               onClose={() => setUnlocked((queue) => queue.slice(1))}/>}

        {ready && unlocked.length === 0 && reveal.closed && <RoundReveal key={revealKeyOf(reveal.closed)}
                                                                         closed={reveal.closed}
                                                                         countryCode={countryState.code}
                                                                         play={sound.play}
                                                                         onClose={reveal.dismiss}/>}

        {award && <BonusAward reward={award}
                              kept={isFirstWin(bonusGuide.guide, award.kind)}
                              onDone={() => {
                                  bonusGuide.markWon(award.kind)
                                  dismissAward()
                              }}/>}

        {ready && connection === "down" && <ConnectionLost lowered={quiz.state.phase !== 'idle'}/>}

        {lastClose && newest === lastClose && <FortifyHint
            key={`close-${lastClose.id}`}
            close={lastClose.close}
            lowered={quiz.state.phase !== 'idle'}
            onDone={dismissClose}
        />}

        {lastFortified && newest === lastFortified && <FortifyNews
            key={`fortify-${lastFortified.id}`}
            fortified={lastFortified.fortified}
            lowered={quiz.state.phase !== 'idle'}
            onDone={dismissFortified}
        />}

        {fortifyCard && <FortifyCard fortified={fortifyCard} onDone={() => {
            fortifyGuide.markSeen()
            setFortifyCard(undefined)
        }}/>}

        {lastBomb && newest === lastBomb && <BombNews
            key={lastBomb.id}
            drop={lastBomb.drop}
            land={lastBomb.land}
            lowered={quiz.state.phase !== 'idle'}
            onDone={dismissBomb}
        />}

        <Quiz state={quiz.state} onOpen={quiz.open} onAnswer={quiz.answer}/>

        {vpnBlocked && <VPNBlockedModal onClose={dismissVPNBlocked}/>}
        {bannedUntil !== undefined && <BannedModal until={bannedUntil} onClose={dismissBanned}/>}

        {sessionUnavailable && <SessionUnavailableModal onClose={dismissSessionUnavailable}/>}
    </>
}

function StatusCard({status}: {status: GlobeStatus}) {
    return <div className="viewer-status">
        <div className="viewer-status-card panel" role="status">
            {status.state === 'loading'
                ? <LoadingCard territories={status.territories}/>
                : status.state === 'failed' && <>
                    <h3>The globe could not be loaded</h3>
                    <p>{status.message}</p>
                </>}
        </div>
    </div>
}

function LoadingCard({territories}: {territories: number | undefined}) {
    const percent = Math.round((territories ?? 0) * 100)
    return <>
        <div className="viewer-status-spinner"/>
        <h3>{territories === undefined ? "Loading the planet…" : "Loading territories…"}</h3>
        <div className="viewer-status-progress"
             role="progressbar"
             aria-label="Territories loaded"
             aria-valuemin={0}
             aria-valuemax={100}
             aria-valuenow={percent}>
            <div className="viewer-status-progress-fill" style={{width: `${percent}%`}}/>
        </div>
    </>
}
