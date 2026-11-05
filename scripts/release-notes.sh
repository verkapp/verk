#!/usr/bin/env bash
# Extrait les notes d'une version depuis CHANGELOG.md.
#
# Usage : scripts/release-notes.sh <version>
#   scripts/release-notes.sh 1.2.4
#   scripts/release-notes.sh v1.2.4
#
# Sort sur la sortie standard le contenu de la section `## [<version>]`
# (sans l'en-tête), et renvoie 1 si la section est absente ou vide.
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
  echo "usage: $(basename "$0") <version>" >&2
  exit 2
fi
version="${version#v}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
changelog="$root/CHANGELOG.md"
if [[ ! -f "$changelog" ]]; then
  echo "CHANGELOG.md introuvable à la racine du dépôt." >&2
  exit 1
fi

notes="$(
  awk -v version="$version" '
    index($0, "## [" version "]") == 1 { found = 1; next }
    found && /^## \[/ { exit }
    found { print }
  ' "$changelog"
)"

# Retire les lignes vides de début et de fin.
notes="$(
  printf '%s\n' "$notes" | awk '
    { lines[NR] = $0 }
    END {
      first = 1
      last = NR
      while (first <= last && lines[first] ~ /^[[:space:]]*$/) first++
      while (last >= first && lines[last] ~ /^[[:space:]]*$/) last--
      for (i = first; i <= last; i++) print lines[i]
    }
  '
)"

if [[ -z "$notes" ]]; then
  echo "Aucune note pour la version $version dans CHANGELOG.md." >&2
  exit 1
fi

printf '%s\n' "$notes"
