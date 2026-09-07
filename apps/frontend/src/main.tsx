import {createRoot} from 'react-dom/client'
import './index.css'

import {
    createBrowserRouter,
    RouterProvider,
} from 'react-router-dom';

import {StrictMode} from "react";
import {ClickServiceClient, HTTPBackend} from "./backends/httpBackend.ts";
import App from "./app/App.tsx";

const clickServiceClient = new ClickServiceClient({
    baseUrl: import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol",
    timeoutMs: 2000
})

const backend = new HTTPBackend(clickServiceClient, 100)
// const backend = new FakeBackend(500)

const router = createBrowserRouter([{
    path: "",
    element: function () {
        return (
            <StrictMode>
                <App
                    ownershipsGetter={backend}
                    tileClicker={backend}
                    updatesListener={backend}
                />
            </StrictMode>
        )
    }()
}], {
    // React Router v6 warns once per flag about behaviour that changes in v7.
    // This app has one route with no loaders, actions or fetchers, so every
    // one of these is a no-op here — opting in only silences the warnings and
    // makes the eventual v7 upgrade a version bump.
    future: {
        v7_relativeSplatPath: true,
        v7_fetcherPersist: true,
        v7_normalizeFormMethod: true,
        v7_partialHydration: true,
        v7_skipActionErrorRevalidation: true,
    },
})

createRoot(document.getElementById('root')!).render(
    // v7_startTransition lives on the provider, not the router: it changes how
    // React schedules the update, not how routes resolve.
    <RouterProvider router={router} future={{v7_startTransition: true}}/>,
);
