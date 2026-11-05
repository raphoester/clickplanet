# The error net

**No handler's raw error reaches the wire, and no module has to remember that.**
`cpbootstrap` wraps `cpconnect.NewErrorInterceptor` around every service it
mounts, outside whatever interceptors the module named: an unrecognised error is
logged once, with its procedure, and answered as `internal error` with the cause
stripped. `TestEveryMountedServiceGetsTheErrorNet` mounts a module that asks for
nothing and pins that it still gets it.

**It lives at the mount and not in a controller because it is a property of the
process**, not of a context. It used to be three near-identical constructors,
one per module, each of which could have been left out of a chain and none of
which any test would have missed.

**That is also why `Mount` takes a builder rather than a handler.** A Connect
interceptor is baked in at `New<Service>Handler`, so there is nothing to wrap
afterwards, and an HTTP middleware is far too late — by then the error is a
serialized response body. Building inside `Mount` is the only place the server
can put anything around every procedure in the process. The module writes a
one-line closure, because the generated constructor takes the service
*interface* while the module holds the concrete type, and no type parameter can
infer that conversion.

**What each module still owns is its own mapping.** A caller error is named by
the part that found it and turned into a code by the handler that knows which
procedure was asked — `click_handler` and `get_map_handler` in planet,
`send_message_handler` for `messages.ErrInvalidMessage`, `create_session_handler` and `sessionv1controller`
for `attestation.ErrAttestationFailed`. The net never sees those, because a `*connect.Error`
is passed through untouched.

The session one keeps a log line of its own: a refused mint is logged at **Info**
there, because the net logs at Error and a refusal is the check doing its job
rather than a fault of this server — on a public endpoint it is the common case.
