import {useEffect, useMemo, useRef, useState} from 'react';
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
    LandmassReader,
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
import {PlayerInfoBackend, PlayerLine, PresenceBackend} from "../../backends/player.ts";
import PlayerCard from "../players/PlayerCard.tsx";
import HolderCard from "../players/HolderCard.tsx";
import TerritoryCard from "../territory/TerritoryCard.tsx";
import {useTerritoryCard} from "../territory/useTerritoryCard.ts";
import {useTerritoryPlayers} from "../territory/useTerritoryPlayers.ts";
import {useTileHolder} from "../players/useTileHolder.ts";
import TitleUnlocked from "../titles/TitleUnlocked.tsx";
import {usePresence} from "../players/usePresence.ts";
import {useRoster} from "../players/useRoster.ts";
import SignInPitchModal from "../account/SignInPitchModal.tsx";
import SignInNudgeModal from "../account/SignInNudgeModal.tsx";
import {useSignInNudge} from "../account/useSignInNudge.ts";
import {useSignInRefill} from "../account/useSignInRefill.ts";
import {LeaderboardEntry, rankOf} from "../../domain/leaderboard.ts";
import {useSeason} from "../season/useSeason.ts";
import SeasonChip, {ChampionChip, SeasonDetails} from "../season/SeasonChip.tsx";
import {useCompact} from "../compact.ts";
import Sheet from "../hud/Sheet.tsx";
import StatusBar from "../hud/StatusBar.tsx";
import TabBar from "../hud/TabBar.tsx";
import MissionsPlace, {MissionsPlaceProps} from "../missions/MissionsPlace.tsx";
import {useMissions} from "../missions/useMissions.ts";
import {Mission, MissionsBackend} from "../../backends/missions.ts";
import {unseen} from "../../domain/missions.ts";
import MissionsDone from "../missions/MissionsDone.tsx";
import {useSeenMissions} from "../missions/useSeenMissions.ts";
import {usePaidMissions} from "../missions/usePaidMissions.ts";
import {boardsOf, StandingsBackend} from "../../backends/standings.ts";
import {SupportBackend} from "../../backends/support.ts";
import {pageContextOf} from "../support/pageContext.ts";
import {BoardStandings, BoardView} from "../standings/BoardViews.tsx";
import {useRace} from "../standings/useRace.ts";
import {useDebriefed} from "../season/useDebriefed.ts";
import RoundReveal from "../season/RoundReveal.tsx";
import {useRoundReveal} from "../season/useRoundReveal.ts";
import {TurnBackend} from "../../backends/turn.ts";
import {Battle, battleKeyOf, battleOf, finaleOf, nextBattleOf} from "../../domain/battle.ts";
import {announcementKeyOf} from "../../domain/announcements.ts";
import {useAnnouncements} from "../announcements/useAnnouncements.ts";
import {useSchedule} from "../battle/useSchedule.ts";
import {useMood} from "../battle/useMood.ts";
import {moodOf} from "../../domain/mood.ts";
import {useBattleReveal} from "../battle/useBattleReveal.ts";
import {useBattleOver} from "../battle/useBattleOver.ts";
import {useSeenBattle} from "../battle/useSeenBattle.ts";
import BattleChip, {BattleDetails} from "../battle/BattleChip.tsx";
import BattleReveal from "../battle/BattleReveal.tsx";
import FinaleTeaser from "../battle/FinaleTeaser.tsx";
import {useFinaleTeaser} from "../battle/useFinaleTeaser.ts";
import BattleOver from "../battle/BattleOver.tsx";
import BattleDebrief from "../battle/BattleDebrief.tsx";
import {briefDescriptionOf} from "../../domain/battleBrief.ts";
import {revealFollowed} from "../../domain/roundReveal.ts";
import {reigningChampionOf} from "../../domain/champion.ts";
import {countryLines, CountryOrder, shownOrder} from "../../domain/race.ts";
import {endedBookOf} from "../../domain/debrief.ts";
import {acceptedClicks} from "./acceptedClicks.ts";
import {ClipsBackend, ClipsNameBackend} from "../../backends/clips.ts";
import ClipNotice from "../clips/ClipNotice.tsx";
import FlagMissionDone from "../missions/FlagMissionDone.tsx";
import {useFlagMissionDone} from "../missions/useFlagMissionDone.ts";
import {useClipNotice} from "../clips/useClipNotice.ts";
import {CrazyGamesPage} from "../../backends/crazyGames.ts";
import {useCrazyGames, useGameplay, useInvites, useRoom} from "../crazygames/useCrazyGames.ts";
import InvitedModal from "../crazygames/InvitedModal.tsx";
import InviteFriends from "../share/InviteFriends.tsx";
import {MUTED_SOUND_SETTINGS} from "../../domain/soundSettings.ts";
import {Halo} from "./atmosphere.ts";
import "../halloween/Halloween.css"
import "./Viewer.css"

const NO_LEADERBOARD: readonly LeaderboardEntry[] = []

// After the wave has run over the land.
const FORTIFY_CARD_DELAY_MS = 1600

type SheetName = "board" | "missions" | "chat" | "you" | "share" | "invite" | "settings" | "season" | "battle" | "clicks"

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
    landmasses?: LandmassReader
    turn?: TurnBackend
    standings?: StandingsBackend
    support?: SupportBackend
    missions?: MissionsBackend
    chargesRefresher?: ChargesRefresher
    clips?: ClipsBackend & ClipsNameBackend
    crazyGames?: CrazyGamesPage
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
    const invitation = useInvites(props.crazyGames, allegiance, handleSetCountry)
    const invite = props.crazyGames?.room.inviteLink
    const pickCountry = async (country: Country) => {
        if (country.code !== countryState.code) await props.allegiance?.pledgeAllegiance(country.code)
        handleSetCountry(country)
    }
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const connection = useConnection(props.connection)
    const crazyGames = useCrazyGames(props.crazyGames)
    const sound = useSound(crazyGames.silenced)
    const display = useDisplaySettings()
    const account = useAccount(props.account)
    const username = account.kind === 'ready' ? account.username : undefined
    const ownAccountId = account.kind === 'ready' ? account.accountId : undefined
    const color = account.kind === 'ready' ? account.color : undefined
    const linked = account.kind === 'ready' && account.me.linked.length > 0

    usePresence(props.presence, {countryCode: countryState.code, username, color})

    const quiz = useQuiz(props.quizMaster, countryState.code, sound.play)
    const announcements = useAnnouncements()
    const {announce, current} = announcements
    const roster = useRoster(props.presence, (title) => announce({kind: "title", title}))
    const schedule = useSchedule(props.turn)
    const season = useSeason(schedule.seasons.current)
    const bookId = schedule.seasons.current?.bookId ?? endedBookOf(schedule.debrief)
    const dayEndsAt = schedule.seasons.today?.endsAt
    const battle = battleOf(schedule.turn)
    const race = useRace(props.standings, bookId, schedule.seasons.today?.roundId, battle?.roundId)
    const closed = useDebriefed(props.standings, schedule.debrief)
    const nextBattle = nextBattleOf(schedule.battles, Date.now())
    const finale = finaleOf(schedule.battles, schedule.turn, Date.now())
    const [accepted] = useState(acceptedClicks)
    const finaleTeaser = useFinaleTeaser(finale, Date.now(), accepted.listenForClicks)
    const mood = moodOf(schedule.turn, clickBudget)
    const halloween = schedule.turn?.kind === "halloween"
    useMood(mood)
    const battleReveal = useBattleReveal(battle)
    const battleOver = useBattleOver(battle, schedule.debrief)
    const seenBattle = useSeenBattle(battle)
    const reveal = useRoundReveal(closed, seenBattle)
    useEffect(() => {
        for (const due of reveal.due) announce(due)
    }, [announce, reveal.due])
    useEffect(() => {
        if (battleReveal.battle) announce({kind: "battle", battle: battleReveal.battle})
    }, [announce, battleReveal.battle])
    useEffect(() => {
        if (finaleTeaser.finale) announce({kind: "finale", finale: finaleTeaser.finale})
    }, [announce, finaleTeaser.finale])
    const clipNotice = useClipNotice(props.clips, linked && username !== undefined ? ownAccountId : undefined)
    useEffect(() => {
        if (clipNotice.appearance) announce({kind: "clip", appearance: clipNotice.appearance})
    }, [announce, clipNotice.appearance])
    const champion = reigningChampionOf(season, closed)
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
    const [battleOpen, setBattleOpen] = useState(false)
    const [clicksOpen, setClicksOpen] = useState(false)
    const [unread, setUnread] = useState(0)
    const [boardView, setBoardView] = useState<BoardView>("countries")
    const toggleSheet = (name: SheetName) => setSheet((current) => current === name ? undefined : name)
    const closeSheet = () => setSheet(undefined)

    useEffect(() => {
        setSheet(undefined)
        setSeasonOpen(false)
        setBattleOpen(false)
        setClicksOpen(false)
    }, [compact])

    const {
        status,
        tilesLoading,
        orbit,
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
        halo: haloOf(battle, halloween),
        bonusLook: halloween ? "pumpkin" : "box",
        rendering: display.settings.rendering,
    })

    useGameplay(props.crazyGames, status.state === 'ready')
    useRoom(props.crazyGames, status.state === 'ready', countryState.code)

    const tileHolder = useTileHolder(props.tileHolders, props.playerInfo, pointing, ownAccountId)
    const territoryCard = useTerritoryCard(pointingTerritory, territoryOf, showTerritory)
    const territoryPlayers = useTerritoryPlayers(props.landmasses, props.playerInfo, territoryCard.shown?.territory.landmass)

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

    const signInRefill = rules?.signInRefill === true
    // CrazyGames forbids a sign-in that pops up or is the main call to action.
    const nudge = useSignInNudge(
        !props.crazyGames && guest && clickBudget?.linkedMultiplier !== undefined && !clickBudget.frozen && !charges.refill && !pitchOpen,
        refusals, quiz.state.phase !== 'idle')
    useSignInRefill(account.kind === 'ready' && account.guestLinked === true && signInRefill, charges.refill,
        () => void props.chargesRefresher?.refreshCharges().catch((e) => console.error("could not read the charges held", e)),
        () => void spendRefill?.())

    const order = shownOrder(race, countryOrder)
    const lines = countryLines(leaderboard, race, order)
    const anthem = useAnthem(status.state === 'ready' ? lines : NO_LEADERBOARD, order, crazyGames.silenced ? MUTED_SOUND_SETTINGS : sound.settings)

    const stats = shareStats(lines, countryState)
    const {shot, taking, take, discard} = useSharePicture(capture, stats)

    const openPitch = guest ? () => {
        closeSheet()
        setClicksOpen(false)
        setPitchOpen(true)
    } : undefined

    const toll = rules?.toll ?? []
    const held = leaderboard.find((entry) => entry.country.code === countryState.code)?.tiles ?? 0
    const boards = useMemo(() => props.standings && bookId ? boardsOf(props.standings, bookId) : undefined, [props.standings, bookId])
    const standings: BoardStandings | undefined = boards && {
        backend: boards,
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
    const seenMissions = useSeenMissions()
    const [missionsDone, setMissionsDone] = useState<Mission[]>()
    usePaidMissions(missionsState, (paid) => {
        setMissionsDone(paid)
        const refresh = () => props.chargesRefresher?.refreshCharges().catch((e) => console.error("could not read the charges held", e))
        refresh()
        setTimeout(refresh, GRANT_LANDS_MS)
    })
    useFlagMissionDone(missionsState, (mission) => {
        if (allegiance) announce({kind: "flagMission", mission, country: allegiance.countryId})
    })
    const missions: MissionsPlaceProps | undefined = props.missions && {
        state: missionsState,
        onSignIn: guest ? openPitch ?? (() => setSheet("you")) : undefined,
        onSeen: seenMissions.markSeen,
    }
    const missionsNews = missionsState.kind === "ready" ? unseen(missionsState.day.missions, seenMissions.seen) : 0

    const you = {
        account: props.account,
        linkedMultiplier: clickBudget?.linkedMultiplier,
        playerInfo: props.playerInfo,
        listenForClicks: accepted.listenForClicks,
        clips: props.clips,
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

    return <>
        <div ref={container} className="viewer-canvas"/>

        {!ready && <StatusCard status={status}/>}
        {ready && !orbit && tilesLoading !== undefined && <StatusCard status={{state: 'loading', territories: tilesLoading}}/>}

        {ready && !compact && <Menu {...board} {...you} {...share} {...settings}
                                    missions={missions}
                                    missionsNews={missionsNews}
                                    invite={invite}
                                    tab={menuTab}
                                    onTab={setMenuTab}/>}

        {ready && !compact && battle && <BattleChip battle={battle}
                                                    season={season}
                                                    compact={false}
                                                    open={battleOpen}
                                                    onToggle={() => setBattleOpen((open) => !open)}/>}
        {ready && !compact && !battle && champion && <ChampionChip champion={champion} compact={false}/>}
        {ready && !compact && !battle && !champion && season && <SeasonChip season={season}
                                                                 nextBattle={nextBattle}
                                                                 finale={finale}
                                                                 dayEndsAt={dayEndsAt}
                                                                 compact={false}
                                                                 open={seasonOpen}
                                                                 onToggle={() => setSeasonOpen((open) => !open)}/>}

        {ready && compact && <StatusBar country={countryState}
                                        rank={rankOf(lines, countryState)}
                                        boardOpen={sheet === "board"}
                                        onOpenBoard={() => toggleSheet("board")}
                                        trailing={battle
                                            ? <BattleChip battle={battle}
                                                          season={season}
                                                          compact
                                                          open={sheet === "battle"}
                                                          onToggle={() => toggleSheet("battle")}/>
                                            : champion
                                                ? <ChampionChip champion={champion} compact/>
                                                : season && <SeasonChip season={season}
                                                                        nextBattle={nextBattle}
                                                                        finale={finale}
                                                                        dayEndsAt={dayEndsAt}
                                                                        compact
                                                                        open={sheet === "season"}
                                                                        onToggle={() => toggleSheet("season")}/>}/>}

        {ready && <ClickBudgetMeter budget={clickBudget}
                                    frozen={mood === "frozen" ? {nextSeason: schedule.seasons.next, country: countryState} : undefined}
                                    invite={invite}
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

        {ready && crazyGames.chat && <ChatPanel
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
            {sheet === "invite" && invite && <Sheet title="Invite" onClose={closeSheet}>
                <InviteFriends country={countryState} invite={invite}/>
            </Sheet>}
            {sheet === "settings" && <Sheet title="Settings" onClose={closeSheet}>
                <SettingsPlace {...settings}/>
            </Sheet>}
            {sheet === "season" && season && <Sheet title={`Season ${season.number}`} onClose={closeSheet}>
                <SeasonDetails season={season} nextBattle={nextBattle} finale={finale} dayEndsAt={dayEndsAt}/>
            </Sheet>}
            {sheet === "battle" && battle && <Sheet title="Battle" onClose={closeSheet}>
                <BattleDetails battle={battle} season={season}/>
            </Sheet>}
            {sheet === "clicks" && <Sheet title="Your clicks" onClose={closeSheet}>{clicks}</Sheet>}

            <TabBar open={sheet}
                    onOpen={toggleSheet}
                    chat={props.chatBackend !== undefined && crazyGames.chat}
                    invite={invite !== undefined}
                    unread={unread}
                    you={account.kind === 'ready' ? (linked ? "player" : "guest") : undefined}
                    missions={missions ? missionsNews : undefined}/>
        </>}

        {shot && <SharePreview shot={shot}
                               stats={stats}
                               onClose={discard}/>}

        {pitchOpen && guest && props.account && clickBudget?.linkedMultiplier && <SignInPitchModal
            state={account}
            store={props.account}
            multiplier={clickBudget.linkedMultiplier}
            onClose={() => setPitchOpen(false)}/>}

        {nudge.open && guest && props.account && clickBudget?.linkedMultiplier && <SignInNudgeModal
            state={account}
            store={props.account}
            multiplier={clickBudget.linkedMultiplier}
            fullBank={signInRefill}
            onClose={nudge.close}/>}

        {invitation.asked && allegiance && <InvitedModal key={invitation.asked.code}
                                                         friend={invitation.asked}
                                                         held={allegiance}
                                                         onSwitch={() => pickCountry(invitation.asked!)}
                                                         onClose={invitation.answer}/>}

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
                                               players={territoryPlayers}
                                               onOpenPlayer={onOpenPlayer && ((player) => {
                                                   territoryCard.dismiss()
                                                   onOpenPlayer(player)
                                               })}
                                               onHold={territoryCard.hold}
                                               onClose={territoryCard.dismiss}/>}

        {openPlayer && props.playerInfo && <PlayerCard key={openPlayer.accountId}
                                                       player={openPlayer}
                                                       backend={props.playerInfo}
                                                       reporter={openPlayer.accountId === ownAccountId ? undefined : props.support}
                                                       onClose={() => setOpenPlayer(undefined)}/>}

        {ready && current?.kind === "title" && <TitleUnlocked key={announcementKeyOf(current)}
                                                              title={current.title}
                                                              play={sound.play}
                                                              onWear={props.account?.wearTitle}
                                                              onClose={announcements.dismiss}/>}

        {ready && (current?.kind === "round" || current?.kind === "results") && <RoundReveal
            key={announcementKeyOf(current)}
            closed={current.closed}
            results={current.kind === "results"}
            countryCode={countryState.code}
            play={sound.play}
            onClose={() => {
                reveal.dismiss(current)
                announcements.dismiss()
            }}/>}

        {ready && (current?.kind === "over" || current?.kind === "points" || current?.kind === "players") && <BattleDebrief
            key={announcementKeyOf(current)}
            screen={current.kind}
            closed={current.closed}
            description={briefDescriptionOf(seenBattle, current.closed)}
            countryCode={countryState.code}
            accountId={ownAccountId}
            more={revealFollowed(current, seenBattle)}
            play={sound.play}
            onClose={() => {
                reveal.dismiss(current)
                announcements.dismiss()
            }}/>}

        {ready && current?.kind === "battle" && <BattleReveal key={announcementKeyOf(current)}
                                                              battle={current.battle}
                                                              play={sound.play}
                                                              onClose={() => {
                                                                  battleReveal.dismiss()
                                                                  announcements.dismiss()
                                                              }}/>}

        {ready && current?.kind === "finale" && <FinaleTeaser key={announcementKeyOf(current)}
                                                              finale={current.finale}
                                                              play={sound.play}
                                                              onClose={() => {
                                                                  finaleTeaser.dismiss(current.finale)
                                                                  announcements.dismiss()
                                                              }}/>}

        {ready && current?.kind === "clip" && <ClipNotice key={announcementKeyOf(current)}
                                                          appearance={current.appearance}
                                                          play={sound.play}
                                                          onClose={() => {
                                                              clipNotice.dismiss(current.appearance)
                                                              announcements.dismiss()
                                                          }}/>}

        {ready && current?.kind === "flagMission" && <FlagMissionDone key={announcementKeyOf(current)}
                                                                      mission={current.mission}
                                                                      country={current.country}
                                                                      play={sound.play}
                                                                      onClose={announcements.dismiss}/>}

        {missionsDone && rules && <MissionsDone missions={missionsDone}
                                                rules={rules}
                                                onDone={() => setMissionsDone(undefined)}/>}

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

        {battleOver.over && <BattleOver key={battleKeyOf(battleOver.over)}
                                        awaiting={battleOver.awaiting}
                                        lowered={quiz.state.phase !== 'idle'}
                                        onDone={battleOver.dismiss}/>}

        <Quiz state={quiz.state} onOpen={quiz.open} onAnswer={quiz.answer}/>

        {vpnBlocked && <VPNBlockedModal onClose={dismissVPNBlocked}/>}
        {bannedUntil !== undefined && <BannedModal until={bannedUntil} onClose={dismissBanned}/>}

        {sessionUnavailable && <SessionUnavailableModal onClose={dismissSessionUnavailable}/>}
    </>
}

function haloOf(battle: Battle | undefined, halloween: boolean): Halo {
    if (battle) return "war"
    if (halloween) return "haunt"
    return "calm"
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
