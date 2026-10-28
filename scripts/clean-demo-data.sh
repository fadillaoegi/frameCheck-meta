#!/usr/bin/env bash
# Removes every project from the local FrameCheck database except the ones kept,
# then deletes stored image files that no remaining asset references.
#
#   scripts/clean-demo-data.sh                 # dry run: show what would go
#   scripts/clean-demo-data.sh --yes           # back up, then delete
#   scripts/clean-demo-data.sh --keep <id> ... # keep these project ids instead
#
# Default keep: the "Sample · Lembar landing page" project with the most runs.
# Stop the API first so no run is in progress. Local data only; never touches .env.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STORAGE="$ROOT/framecheck_BE/storage"
APPLY=0
KEEP=()
while [ $# -gt 0 ]; do
  case "$1" in
    --yes) APPLY=1 ;;
    --keep) shift; KEEP+=("$1") ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done

psql() { docker compose --project-directory "$ROOT" --project-name framecheck exec -T postgres psql -U framecheck -d framecheck -v ON_ERROR_STOP=1 -qAt "$@"; }

if [ ${#KEEP[@]} -eq 0 ]; then
  KEEP+=("$(psql -c "SELECT p.id FROM projects p LEFT JOIN runs r ON r.project_id = p.id
                     WHERE p.name = 'Sample · Lembar landing page'
                     GROUP BY p.id, p.created_at ORDER BY count(r.id) DESC, p.created_at LIMIT 1")")
fi
for id in "${KEEP[@]}"; do
  [[ "$id" =~ ^[0-9a-f-]{36}$ ]] || { echo "not a project id: '$id'" >&2; exit 2; }
done
KEEP_SQL="'$(IFS=,; echo "${KEEP[*]}" | sed "s/,/','/g")'"

echo "Keeping:"
psql -c "SELECT '  ' || name || '  (' || id || ')' FROM projects WHERE id IN ($KEEP_SQL) ORDER BY created_at"
echo "Removing:"
psql -c "SELECT '  ' || name || '  (' || id || ')' FROM projects WHERE id NOT IN ($KEEP_SQL) ORDER BY created_at"

if psql -c "SELECT 1 FROM runs WHERE status IN ('queued','running') LIMIT 1" | grep -q 1; then
  echo "A run is still queued or running. Stop the API and try again." >&2
  exit 1
fi

if [ "$APPLY" -ne 1 ]; then
  echo
  echo "Dry run. Re-run with --yes to back up and delete."
  exit 0
fi

BACKUP="$STORAGE/backups/framecheck-$(date +%Y%m%d-%H%M%S).sql"
mkdir -p "$(dirname "$BACKUP")"
docker compose --project-directory "$ROOT" --project-name framecheck exec -T postgres pg_dump -U framecheck -d framecheck > "$BACKUP"
echo "Backup: $BACKUP"

psql <<SQL
BEGIN;
CREATE TEMP TABLE gone AS SELECT id FROM projects WHERE id NOT IN ($KEEP_SQL);
DELETE FROM reviews  WHERE finding_id IN (SELECT f.id FROM findings f JOIN runs r ON r.id = f.run_id WHERE r.project_id IN (SELECT id FROM gone));
DELETE FROM findings WHERE run_id IN (SELECT id FROM runs WHERE project_id IN (SELECT id FROM gone));
DELETE FROM runs                 WHERE project_id IN (SELECT id FROM gone);
DELETE FROM comparisons          WHERE project_id IN (SELECT id FROM gone);
DELETE FROM requirement_versions WHERE project_id IN (SELECT id FROM gone);
DELETE FROM assets               WHERE project_id IN (SELECT id FROM gone);
DELETE FROM projects             WHERE id IN (SELECT id FROM gone);
COMMIT;
SQL

# Image files are content-addressed and may be shared; keep any still referenced.
KEPT_KEYS="$(psql -c "SELECT DISTINCT storage_key FROM assets")"
removed=0
if [ -d "$STORAGE/assets" ]; then
  while IFS= read -r f; do
    key="${f#"$STORAGE"/}"
    if ! grep -qxF "$key" <<<"$KEPT_KEYS"; then
      rm -f "$f"
      removed=$((removed + 1))
    fi
  done < <(find "$STORAGE/assets" -type f)
fi
echo "Deleted projects: done. Deleted unreferenced image files: $removed."
