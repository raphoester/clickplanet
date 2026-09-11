/**
 * The one fact about the page's display face that anything drawing beside its
 * capitals has to know.
 *
 * **Luckiest Guy's capitals do not sit in the middle of its em box.** The face
 * carries far more ascent than they use, so anything centred on the box — a
 * flag, an icon, a canvas `textBaseline: "middle"` — comes out below the letters
 * it is meant to line up with. Every place that puts something next to a title
 * has hit this: the leaderboard's flags, the menu header's, the chat log's, the
 * camera button's icon, and the country name on the share card.
 *
 * `CountryFlag` is the component that settles it for the DOM, and every flag
 * beside a name goes through it. What cannot is the share card, which is drawn
 * on a canvas — so the number lives here rather than in either of them.
 */

export const TITLE_FONT_FAMILY = '"Luckiest Guy", sans-serif'

/** Cap height as a fraction of the em, measured off the face itself. */
export const TITLE_CAP_HEIGHT = 0.716

/**
 * How far above the middle of a `line-height: 1` box the capitals sit, in em.
 *
 * Measured in the browser at 18px: the cap band's centre is 2.32px above the
 * line box's, so anything centred on the box with `align-items: center` needs
 * pulling up by this much to meet the letters. In em so it follows the size.
 *
 * **A flex `margin-top` that does the pulling has to be twice this**, because
 * `align-items: center` centres the margin box: taking 2R off that box moves
 * what is inside it by R.
 */
export const TITLE_CAP_RISE = 0.13
