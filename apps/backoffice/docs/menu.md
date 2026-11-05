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
- **A page picked opens at its top**: Players at its list, Reports with none open. A name on a report or in the
  chat, and a note's "Open player", go to Players with its account open (`OpenPlayer`).
- **The page shown and what is open on it are in the address bar** (`#/reports/report-3`, `#/players/<account id or
  address>`; `routeOf`, `hashOf`, `useRoute`): a reload or a link opens them again, and Back goes to the route
  before, a player's Back to the list as it was. An address that names no page opens Home. The address bar keeps a
  player's id alone; what the page that opened it knew (its name, its row) is kept beside it while it stays open.
