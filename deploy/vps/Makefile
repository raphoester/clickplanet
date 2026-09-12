.PHONY: env env-preview

# Where render-env.sh writes. The deploy workflow renders into the runner's
# workspace and copies the result to the box; overriding this is what lets the
# same target be run locally against a fake SECRETS_JSON.
ENV_OUT ?= .env

# Renders .env from env.public plus every repository secret. SECRETS_JSON is
# filled by the workflow from toJSON(secrets); adding an application secret is
# done entirely in the GitHub UI and needs no edit here.
env:
	@./render-env.sh $(ENV_OUT)

# The same render with invented values, to check the shape of the file without
# holding a single real secret. Useful after editing env.public or the script.
# Writes to a temp file rather than stdout because the script renames its output
# into place, and renaming onto /dev/stdout replaces the device node.
env-preview:
	@tmp=$$(mktemp); \
		SECRETS_JSON='{"EXAMPLE_SECRET":"not-a-real-value","VPS_SSH_KEY":"excluded"}' \
			./render-env.sh "$$tmp" >/dev/null; \
		cat "$$tmp"; rm -f "$$tmp"
