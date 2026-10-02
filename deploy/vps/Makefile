.PHONY: env env-preview

ENV_OUT ?= .

env:
	@./render-env.sh $(ENV_OUT)

env-preview:
	@dir=$$(mktemp -d); \
		SECRETS_JSON='{"CLOUDFLARE_API_TOKEN":"cf-not-real","CHAT_TAG_SALT":"salt-not-real","SESSION_SECRET":"sess-not-real","VPS_SSH_KEY":"excluded"}' \
			./render-env.sh "$$dir"; \
		for f in "$$dir/.env" "$$dir/.env.caddy" "$$dir/.env.backend"; do \
			echo; echo "=== $$(basename $$f) ==="; cat "$$f"; \
		done; \
		rm -rf "$$dir"
