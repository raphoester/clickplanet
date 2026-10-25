# The menu

**A burger at the top left opens a drawer from the left** (`components/Menu`) below 1024px: every page under its
group, the page shown marked. **From 1024px the same list stays open in a sidebar** (`Sidebar`), with no burger,
so a page is one click away (`useWide`, which follows the window as it is resized). `MENU` in `navigation.ts` is
the only list of pages, and a page's label is its title (`titleOf`), shown above it. `Console` in `App.tsx` shows
the page picked.

- **The sidebar stays in place while the page scrolls**, and is not modal: nothing traps the focus or closes it.
- **The drawer is modal**: it takes the focus to the page shown, keeps Tab inside it, and closes on Escape, its
  close button, a click beside it or a page picked, giving the focus back to the burger. The page behind does not
  scroll while it is open.
- **Picking Players while on it goes back to the list.** A report's "Every tool →" and a note's "Open player" go to
  Players with its account open (`OpenPlayer`).
- **The page shown is in the address bar** (`#/reports`, `usePage`): a reload or a link opens it again, and Back
  goes to the page before. An address that names no page opens Home. Only the page is kept, not the
  player open on it.
