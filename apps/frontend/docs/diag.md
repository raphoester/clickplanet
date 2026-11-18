# The page's report on a phone: `?diag`

A phone has no console. When `/play` stays blank on one, open it with `?diag` (`clickplanet.lol/play?diag`) in the
same browser, normal tab: 10 seconds after the load, a white panel covers the page with what a console would tell:

- the address, the browser's user agent, the screen and its pixel ratio;
- how many elements React drew in `#root` (0: the app never drew, or a render threw and React removed it);
- the canvas's size and whether its WebGL context is lost;
- every error, unhandled rejection and `console.error` since the load, with its stack;
- every localStorage and sessionStorage key with its length and the start of its value. A key that names a session,
  a token or an account shows its length alone: the panel is screenshotted and sent around.

A tap on the panel removes it. Without `?diag`, `src/diag.ts` does nothing. It is imported first in `main.tsx`, so it
hears the errors of every module after it.
