#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 <new-module-name>"
	echo "Example: $0 github.com/me/myproject"
	echo ""
	echo "Renames the Go module from 'brook' to <new-module-name>, and renames"
	echo "the project's non-Go 'brook' references (SQLite file, Badger dir,"
	echo "swagger title) to the"
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

# 8. Update embedded-store project naming: SQLite file, Badger dir, and the
#    Makefile migrate targets across .env.example and CI.
for f in .env.example .github/workflows/ci.yml; do
		[ -f "$f" ] || continue
		sed -i \
			-e "s|^SQLITE_DSN=$old_name\.db|SQLITE_DSN=$new_name.db|" \
			-e "s|^BADGER_DIR=$old_name|BADGER_DIR=$new_name|" \
			"$f"
	done
echo "  updated .env.example, .github/workflows/ci.yml"

# 8. Update SQLite/Badger placeholders in config/*.yaml
for f in config/config_dev.yaml config/config_prd.yaml; do
	[ -f "$f" ] || continue
	sed -i \
		-e "s|dsn: \"$old_name\.db\"|dsn: \"$new_name.db\"|" \
		-e "s|dir: \"$old_name\"|dir: \"$new_name\"|" \
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
