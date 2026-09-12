.PHONY: env env-preview

# Where render-env.sh writes .env and .env.backend. The deploy workflow renders
# into the runner's workspace and copies both to the box; overriding this is
# what lets the same target run locally against a fake SECRETS_JSON.
ENV_OUT ?= .

# Renders both env files, each from the secrets its own service's config names:
# the Caddyfile's {env.NAME} into .env, backend.yaml's env://NAME into
# .env.backend. Adding a secret is the GitHub UI plus the line that uses it.
env:
	@./render-env.sh $(ENV_OUT)

# The same render with invented values, to check the shape of both files without
# holding a single real secret.
env-preview:
	@dir=$$(mktemp -d); \
		SECRETS_JSON='{"CLOUDFLARE_API_TOKEN":"cf-not-real","CHAT_TAG_SALT":"salt-not-real","SESSION_SECRET":"sess-not-real","VPS_SSH_KEY":"excluded"}' \
			./render-env.sh "$$dir"; \
		for f in "$$dir/.env" "$$dir/.env.backend"; do \
			echo; echo "=== $$(basename $$f) ==="; cat "$$f"; \
		done; \
		rm -rf "$$dir"
