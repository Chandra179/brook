#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 <new-module-name>"
	echo "Example: $0 github.com/me/myproject"
	echo ""
	echo "Renames the Go module from 'brook' to <new-module-name>, and renames"
	echo "the project's non-Go 'brook' references (Docker service name, Postgres"
	echo "user/password/db, application_name in config, swagger title) to the"
	echo "last path segment of <new-module-name>."
	exit 1
}

if [ $# -ne 1 ]; then
	usage
fi

old="brook"
new="$1"
old_name="brook"
new_name="${new##*/}"
old_title="Brook"
new_title="$(tr '[:lower:]' '[:upper:]' <<<"${new_name:0:1}")${new_name:1}"

if [ "$old" = "$new" ]; then
	echo "Old and new module names are identical. Nothing to do."
	exit 0
fi

echo "Renaming module '$old' -> '$new' (project name '$old_name' -> '$new_name')"

# 1. Update go.mod module line
sed -i "s|^module $old\$|module $new|" go.mod
echo "  updated go.mod"

# 2. Update import paths in all .go files
find . -name '*.go' -exec sed -i "s|\"$old/|\"$new/|g" {} +
echo "  updated import paths in .go files"

# 3. Also check .proto files if any
if find . -name '*.proto' -print0 | xargs -0 grep -l "$old/" 2>/dev/null; then
	find . -name '*.proto' -exec sed -i "s|$old/|$new/|g" {} +
	echo "  updated .proto files"
fi

# 4. Update local-prefixes in .golangci.yml (YAML list: "- brook")
sed -i "s|^\(\s*- \)$old\$|\1$new|" .golangci.yml
echo "  updated .golangci.yml"

# 5. Update mockery's packages key, which mirrors the full module path
sed -i "s|^\(\s*\)$old:$|\1$new:|" .mockery.yaml
echo "  updated .mockery.yaml"

# 6. Update the swagger @title annotation (source of truth for docs/, which
#    is generated — run `make swag` after this script to regenerate it)
sed -i "s|@title           $old_title API|@title           $new_title API|" cmd/example/main.go
echo "  updated swagger @title in cmd/example/main.go (run 'make swag' to regenerate docs/)"

# 7. Rename the entrypoint directory cmd/example -> cmd/<new_name>.
if [ -d "cmd/example" ]; then
	mv cmd/example "cmd/$new_name"
	echo "  renamed cmd/example -> cmd/$new_name"
fi

# 8. Update Docker/Postgres project naming: service name, user/password/db,
#    and DSNs across docker-compose.yml, .env.example, and CI.
for f in docker-compose.yml .env.example .github/workflows/ci.yml; do
		[ -f "$f" ] || continue
		sed -i \
			-e "s|^  $old_name:$|  $new_name:|" \
			-e "s|postgres://$old_name:$old_name@|postgres://$new_name:$new_name@|g" \
			-e "s|/$old_name?sslmode|/$new_name?sslmode|g" \
			-e "s|POSTGRES_USER: $old_name|POSTGRES_USER: $new_name|" \
			-e "s|POSTGRES_PASSWORD: $old_name|POSTGRES_PASSWORD: $new_name|" \
			-e "s|POSTGRES_DB: $old_name|POSTGRES_DB: $new_name|" \
			-e "s|ARANGO_ROOT_PASSWORD: $old_name|ARANGO_ROOT_PASSWORD: $new_name|" \
			-e "s|ARANGODB_PASSWORD=$old_name|ARANGODB_PASSWORD=$new_name|" \
			-e "s|ARANGODB_DATABASE=$old_name|ARANGODB_DATABASE=$new_name|" \
			"$f"
	done
echo "  updated docker-compose.yml, .env.example, .github/workflows/ci.yml"

# 8. Update application_name and DSN placeholders in config/*.yaml
for f in config/config_dev.yaml config/config_prd.yaml; do
	[ -f "$f" ] || continue
	sed -i \
		-e "s|application_name: \"$old_name\.dev\"|application_name: \"$new_name.dev\"|" \
		-e "s|application_name: \"$old_name\"|application_name: \"$new_name\"|" \
		-e "s|postgres://$old_name:$old_name@|postgres://$new_name:$new_name@|g" \
		-e "s|/$old_name?sslmode|/$new_name?sslmode|g" \
		-e "s|database: \"$old_name\"|database: \"$new_name\"|" \
		"$f"
done
echo "  updated config/config_dev.yaml, config/config_prd.yaml"

# 9. Update Makefile references to the entrypoint directory.
sed -i \
	-e "s|cmd/example/main.go|cmd/$new_name/main.go|g" \
	-e "s|./cmd/example/|./cmd/$new_name/|g" \
	Makefile
echo "  updated Makefile entrypoint references"

echo "Done. Run 'go build ./...' to verify, 'make swag' to regenerate docs/,"
echo "and review README.md/CLAUDE.md/AGENTS.md, which describe 'brook' in prose"
echo "and are not rewritten by this script."
