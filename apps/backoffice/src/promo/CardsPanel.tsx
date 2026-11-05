import {JsonValue} from "@bufbuild/protobuf"
import {useRef, useState} from "react"
import {PromoAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {GetRulesResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Background, Card, CardDraft, ExampleBattle, GetCardsResponse, Placeholder, SetCards} from "../gen/grpc/promo/v1/admin_pb.ts"
import {DEFAULT_SET} from "../ops/rules.ts"
import {draftProblems, MAX_CARDS, MAX_HEADLINE, MAX_LINE, playedBySet, saveRefusalOf} from "./actions.ts"
import {fill, namesBattle, Values} from "./fill.ts"

type Props = {
    promo: PromoAdmin
    library: Background[]
    cards: GetCardsResponse
    rules: GetRulesResponse
    onSaved: () => void
}

export function CardsPanel({promo, library, cards, rules, onSaved}: Props) {
    const first = cards.sets.find(set => set.cards.length > 0) ?? cards.sets.find(set => set.ruleSet !== DEFAULT_SET) ?? cards.sets[0]
    const [chosen, setChosen] = useState(first?.ruleSet ?? "")
    const set = cards.sets.find(each => each.ruleSet === chosen)
    const played = playedBySet(rules)

    return (
        <section className="panel" aria-label="Cards">
            <div className="panel-head">
                <h2>Cards</h2>
                <label className="field promo-set">
                    <span className="muted">Rule set</span>
                    <select value={chosen} onChange={event => setChosen(event.target.value)}>
                        {cards.sets.map(each => (
                            <option key={each.ruleSet} value={each.ruleSet}>
                                {each.ruleSet} · {each.cards.length === 0 ? "no cards" : `${each.cards.length} cards`}
                            </option>
                        ))}
                    </select>
                </label>
            </div>
            <p className="muted">
                A battle that plays this set is teased with these cards, in this order. A set with none is not teased.
            </p>
            {set === undefined
                ? <p className="muted">Planet keeps no rule set.</p>
                : (
                    <SetEditor
                        key={set.ruleSet}
                        promo={promo}
                        set={set}
                        library={library}
                        placeholders={cards.placeholders}
                        example={cards.exampleBattle ?? new ExampleBattle()}
                        played={played.get(set.ruleSet)}
                        onSaved={onSaved}
                    />
                )}
        </section>
    )
}

type Field = "headline" | "line"
type Cursor = {field: Field, start: number, end: number}

function SetEditor({promo, set, library, placeholders, example, played, onSaved}: {
    promo: PromoAdmin, set: SetCards, library: Background[], placeholders: Placeholder[], example: ExampleBattle,
    played: JsonValue | undefined, onSaved: () => void,
}) {
    const [drafts, setDrafts] = useState<CardDraft[]>(() => set.cards.map(card => new CardDraft(card.draft)))
    const values: Values = {played, rules: placeholders.map(placeholder => placeholder.path), battle: example.placeholders}
    const problems = drafts.map(draft => draftProblems(draft, values))
    const changed = !sameDrafts(drafts, set.cards.map(card => card.draft ?? new CardDraft()))
    const ready = changed && problems.every(each => each.length === 0)

    const edit = (index: number, change: Partial<CardDraft>) =>
        setDrafts(was => was.map((draft, at) => at === index ? new CardDraft({...draft, ...change}) : draft))
    const move = (index: number, by: number) => setDrafts(was => {
        const moved = [...was]
        const [card] = moved.splice(index, 1)
        moved.splice(index + by, 0, card)
        return moved
    })

    return (
        <div className="promo-editor">
            <ol className="cards promo-cards">
                {drafts.map((draft, index) => (
                    <CardEditor
                        // The position is the card's identity: moving one moves its fields with it.
                        key={index}
                        index={index}
                        count={drafts.length}
                        draft={draft}
                        saved={set.cards[index]}
                        library={library}
                        placeholders={placeholders}
                        example={example}
                        values={values}
                        problems={problems[index]}
                        onEdit={change => edit(index, change)}
                        onMove={by => move(index, by)}
                        onRemove={() => setDrafts(was => was.filter((_, at) => at !== index))}
                    />
                ))}
            </ol>
            <div className="toolbar">
                <button
                    type="button"
                    className="button"
                    disabled={drafts.length >= MAX_CARDS}
                    onClick={() => setDrafts(was => [...was, new CardDraft({backgroundId: library[0]?.backgroundId ?? ""})])}
                >
                    Add a card
                </button>
                <span className="muted">{drafts.length} of {MAX_CARDS}</span>
                <div className="toolbar-end">
                    <ConfirmAction
                        label="Save"
                        disabled={!ready}
                        question={drafts.length === 0
                            ? `Remove every card of ${set.ruleSet}? Its battles are not teased then.`
                            : `Save the ${drafts.length === 1 ? "card" : `${drafts.length} cards`} of ${set.ruleSet}?`}
                        run={() => promo.saveCards({ruleSet: set.ruleSet, cards: drafts})}
                        said={() => "Saved. Players read them within a minute."}
                        refusal={saveRefusalOf}
                        onDone={onSaved}
                    />
                </div>
            </div>
        </div>
    )
}

function CardEditor({index, count, draft, saved, library, placeholders, example, values, problems, onEdit, onMove, onRemove}: {
    index: number, count: number, draft: CardDraft, saved: Card | undefined, library: Background[],
    placeholders: Placeholder[], example: ExampleBattle, values: Values, problems: string[],
    onEdit: (change: Partial<CardDraft>) => void, onMove: (by: number) => void, onRemove: () => void,
}) {
    const cursor = useRef<Cursor>({field: "headline", start: draft.headline.length, end: draft.headline.length})
    const remember = (field: Field) => (event: {currentTarget: HTMLInputElement}) => {
        const {selectionStart, selectionEnd, value} = event.currentTarget
        cursor.current = {field, start: selectionStart ?? value.length, end: selectionEnd ?? value.length}
    }
    // A placeholder goes where the operator's cursor was, in the field it was in.
    const insert = (path: string) => {
        const {field, start, end} = cursor.current
        const words = draft[field]
        const token = `{${path}}`
        onEdit({[field]: words.slice(0, start) + token + words.slice(end)})
        cursor.current = {field, start: start + token.length, end: start + token.length}
    }

    // A card kept as it was saved shows what the server fills; a card being written is filled here.
    const unchanged = saved?.draft !== undefined && new CardDraft(saved.draft).equals(draft)
    const background = library.find(each => each.backgroundId === draft.backgroundId)
    const shown = unchanged && saved.filled !== undefined
        ? {headline: saved.filled.headline, line: saved.filled.line}
        : {headline: fill(draft.headline, values).text, line: fill(draft.line, values).text}
    const said = unchanged && saved.problem !== "" ? [saved.problem] : problems
    const name = `Card ${index + 1}`

    return (
        <li className="card promo-card-editor" aria-label={name}>
            <div className="card-head">
                <strong>{name}</strong>
                <span className="promo-card-buttons">
                    <button type="button" className="button-mini" disabled={index === 0} onClick={() => onMove(-1)}
                            aria-label={`Move ${name} up`}>↑</button>
                    <button type="button" className="button-mini" disabled={index === count - 1} onClick={() => onMove(1)}
                            aria-label={`Move ${name} down`}>↓</button>
                    <button type="button" className="button-mini" onClick={onRemove}>Remove</button>
                </span>
            </div>
            <div className="promo-card-body">
                <figure className="promo-shown">
                    <div className="promo-preview" aria-label={`${name} as players read it`}>
                        {background !== undefined && <img src={background.url} alt=""/>}
                        <div className="promo-preview-words">
                            <strong>{shown.headline}</strong>
                            {shown.line !== "" && <span>{shown.line}</span>}
                        </div>
                    </div>
                    {namesBattle(draft.headline + draft.line, example.placeholders) && (
                        <figcaption className="muted">Example: a ×{example.times} battle</figcaption>
                    )}
                </figure>
                <div className="fields">
                    <label className="field">
                        <span className="muted">Background</span>
                        <select value={draft.backgroundId} onChange={event => onEdit({backgroundId: event.target.value})}>
                            {background === undefined && <option value={draft.backgroundId}>Pick a background</option>}
                            {library.map(each => <option key={each.backgroundId} value={each.backgroundId}>{each.name}</option>)}
                        </select>
                    </label>
                    <label className="field">
                        <span className="muted">Headline · {[...draft.headline].length}/{MAX_HEADLINE}</span>
                        <input
                            value={draft.headline}
                            onChange={event => {
                                onEdit({headline: event.target.value})
                                remember("headline")(event)
                            }}
                            onSelect={remember("headline")}
                            onFocus={remember("headline")}
                        />
                    </label>
                    <label className="field">
                        <span className="muted">Line · {[...draft.line].length}/{MAX_LINE}</span>
                        <input
                            value={draft.line}
                            onChange={event => {
                                onEdit({line: event.target.value})
                                remember("line")(event)
                            }}
                            onSelect={remember("line")}
                            onFocus={remember("line")}
                        />
                    </label>
                    <label className="field">
                        <span className="muted">Insert a value at the cursor</span>
                        <select
                            value=""
                            aria-label={`Insert a value in ${name}`}
                            onChange={event => {
                                if (event.target.value !== "") insert(event.target.value)
                            }}
                        >
                            <option value="">A value…</option>
                            <optgroup label="The rules">
                                {placeholders.map(placeholder => (
                                    <option key={placeholder.path} value={placeholder.path}>
                                        {placeholder.path}{placeholder.example !== "" ? ` (default: ${placeholder.example})` : ""}
                                    </option>
                                ))}
                            </optgroup>
                            <optgroup label={`The battle (example: ×${example.times})`}>
                                {example.placeholders.map(placeholder => (
                                    <option key={placeholder.path} value={placeholder.path}>
                                        {placeholder.path}{placeholder.example !== "" ? ` (×${example.times}: ${placeholder.example})` : ""}
                                    </option>
                                ))}
                            </optgroup>
                        </select>
                    </label>
                    {said.length > 0 && (
                        <ul className="promo-problems" role="alert">
                            {said.map(problem => <li key={problem}>{problem}</li>)}
                        </ul>
                    )}
                </div>
            </div>
        </li>
    )
}

function sameDrafts(a: CardDraft[], b: CardDraft[]): boolean {
    return a.length === b.length && a.every((draft, index) => draft.equals(b[index]))
}
