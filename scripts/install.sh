#!/usr/bin/env bash
# Installe Verk sur macOS ou Linux.
#
# Usage :
#   scripts/install.sh               # dernière version publiée
#   scripts/install.sh v3.0.0        # une version précise
#   curl -fsSL <url du script> | bash -s -- v3.0.0
#
# macOS : télécharge Verk-macos-universal.zip depuis la GitHub Release, vérifie
#   son empreinte SHA-256 et installe Verk.app dans /Applications (ou
#   ~/Applications sans droit d'écriture). VERK_INSTALL_DIR choisit un autre dossier.
# Linux : la CI ne publie pas de binaire Linux ; le script compile la version
#   demandée depuis les sources (Go, Node.js, Wails, GTK 4 et WebKitGTK 6.0),
#   puis installe ~/.local/bin/verk et l'entrée du menu des applications.
#
# Le dépôt est privé : le téléchargement passe par la CLI gh (connectée), sinon
# par un jeton GITHUB_TOKEN (ou GH_TOKEN), sinon par les liens publics.
set -euo pipefail

REPO="verkapp/verk"
APP="Verk"
WAILS_VERSION="v3.1.0"

version="${1:-${VERK_VERSION:-latest}}"
if [[ "$version" != "latest" && "$version" != v* ]]; then
  version="v$version"
fi
token="${GITHUB_TOKEN:-${GH_TOKEN:-}}"

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mattention :\033[0m %s\n' "$*" >&2; }
die() {
  printf '\033[1;31merreur :\033[0m %s\n' "$*" >&2
  exit 1
}
have() { command -v "$1" >/dev/null 2>&1; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

gh_ready() { have gh && gh auth status >/dev/null 2>&1; }

# json_field <fichier> <expression> : lit un champ JSON avec jq ou python3.
json_field() {
  if have jq; then
    jq -r "$2" "$1"
  elif have python3; then
    python3 - "$1" "$2" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
expr = sys.argv[2]
if expr == ".tag_name":
    print(data.get("tag_name", ""))
else:  # .assets[] | select(.name == "X") | .id
    name = expr.split('"')[1]
    print(next((a["id"] for a in data.get("assets", []) if a["name"] == name), ""))
PY
  else
    die "jq ou python3 est nécessaire pour lire la réponse de l'API GitHub."
  fi
}

# api_release : écrit le JSON de la release demandée dans $work/release.json.
api_release() {
  [[ -f "$work/release.json" ]] && return
  local path="releases/latest"
  [[ "$version" != "latest" ]] && path="releases/tags/$version"
  curl -fsSL -H "Authorization: Bearer $token" -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/$REPO/$path" -o "$work/release.json" ||
    die "release $version introuvable sur $REPO (jeton valide ?)."
}

# resolve_tag : nom du tag à installer (v1.2.3).
resolve_tag() {
  if [[ "$version" != "latest" ]]; then
    echo "$version"
  elif gh_ready; then
    gh release view --repo "$REPO" --json tagName -q .tagName
  elif [[ -n "$token" ]]; then
    api_release
    json_field "$work/release.json" ".tag_name"
  else
    git ls-remote --tags --refs "https://github.com/$REPO.git" 'v*' |
      sed 's|.*refs/tags/||' | sort -V | tail -n 1
  fi
}

# download <asset> : télécharge un fichier de la release dans $work.
download() {
  local name="$1"
  if gh_ready; then
    local args=(release download --repo "$REPO" --pattern "$name" --dir "$work" --clobber)
    [[ "$version" != "latest" ]] && args=(release download "$version" --repo "$REPO" --pattern "$name" --dir "$work" --clobber)
    gh "${args[@]}" || die "téléchargement de $name impossible avec gh."
  elif [[ -n "$token" ]]; then
    api_release
    local id
    id="$(json_field "$work/release.json" ".assets[] | select(.name == \"$name\") | .id")"
    [[ -n "$id" ]] || die "$name absent de la release."
    curl -fsSL -H "Authorization: Bearer $token" -H "Accept: application/octet-stream" \
      "https://api.github.com/repos/$REPO/releases/assets/$id" -o "$work/$name" ||
      die "téléchargement de $name impossible."
  else
    local url="https://github.com/$REPO/releases/latest/download/$name"
    [[ "$version" != "latest" ]] && url="https://github.com/$REPO/releases/download/$version/$name"
    curl -fsSL "$url" -o "$work/$name" ||
      die "téléchargement de $name impossible. Le dépôt est privé : connecte gh (gh auth login) ou exporte GITHUB_TOKEN."
  fi
}

verify() {
  local file="$1" expected actual
  expected="$(awk '{print tolower($1); exit}' "$file.sha256")"
  if have shasum; then
    actual="$(shasum -a 256 "$file" | awk '{print $1}')"
  else
    actual="$(sha256sum "$file" | awk '{print $1}')"
  fi
  [[ -n "$expected" && "$expected" == "$actual" ]] ||
    die "empreinte SHA-256 invalide pour $(basename "$file")."
  info "Empreinte SHA-256 vérifiée."
}

install_macos() {
  local asset="$APP-macos-universal.zip"
  info "Téléchargement de $asset ($version)…"
  download "$asset"
  download "$asset.sha256"
  verify "$work/$asset"

  local dest="${VERK_INSTALL_DIR:-/Applications}"
  if [[ -z "${VERK_INSTALL_DIR:-}" && ! -w "$dest" ]]; then
    dest="$HOME/Applications"
  fi
  mkdir -p "$dest"

  # Ferme seulement la copie que l'on remplace.
  if pgrep -f "$dest/$APP.app/" >/dev/null 2>&1; then
    info "Fermeture de $APP…"
    osascript -e "quit app \"$dest/$APP.app\"" >/dev/null 2>&1 || true
    sleep 2
  fi

  ditto -x -k "$work/$asset" "$work/app"
  [[ -d "$work/app/$APP.app" ]] || die "$APP.app absent de l'archive."
  rm -rf "${dest:?}/$APP.app"
  ditto "$work/app/$APP.app" "$dest/$APP.app"
  # L'app n'est pas notarisée : on retire la quarantaine pour éviter le blocage Gatekeeper.
  xattr -dr com.apple.quarantine "$dest/$APP.app" 2>/dev/null || true

  info "$APP est installé dans $dest/$APP.app"
  echo "    Lance-le avec : open \"$dest/$APP.app\""
}

linux_missing_deps() {
  local missing=()
  for tool in git go npm pkg-config; do
    have "$tool" || missing+=("$tool")
  done
  have cc || have gcc || have clang || missing+=("compilateur C (gcc)")
  if have pkg-config; then
    pkg-config --exists gtk4 || missing+=("GTK 4 (dev)")
    pkg-config --exists webkitgtk-6.0 || missing+=("WebKitGTK 6.0 (dev)")
  fi
  printf '%s\n' "${missing[@]+"${missing[@]}"}"
}

install_linux() {
  local missing
  missing="$(linux_missing_deps)"
  if [[ -n "$missing" ]]; then
    warn "dépendances manquantes :"
    while IFS= read -r dep; do
      printf '  - %s\n' "$dep" >&2
    done <<<"$missing"
    cat >&2 <<'EOF'

Installe-les, par exemple :
  Debian / Ubuntu : sudo apt install git golang nodejs npm pkg-config build-essential libgtk-4-dev libwebkitgtk-6.0-dev
  Fedora          : sudo dnf install git golang nodejs npm pkgconf gcc gtk4-devel webkitgtk6.0-devel
  Arch            : sudo pacman -S git go nodejs npm pkgconf base-devel gtk4 webkitgtk-6.0
Go 1.26+ et Node.js 22+ sont requis ; les versions des dépôts peuvent être trop anciennes.
EOF
    exit 1
  fi

  local tag
  tag="$(resolve_tag)"
  [[ -n "$tag" ]] || die "aucune version publiée trouvée sur $REPO."
  info "Compilation de $APP $tag depuis les sources (quelques minutes)…"

  local src="$work/src"
  if gh_ready; then
    gh repo clone "$REPO" "$src" -- --quiet --depth 1 --branch "$tag"
  elif [[ -n "$token" ]]; then
    git -c "http.extraHeader=Authorization: Basic $(printf 'x-access-token:%s' "$token" | base64 | tr -d '\n')" \
      clone --quiet --depth 1 --branch "$tag" "https://github.com/$REPO.git" "$src"
  else
    git clone --quiet --depth 1 --branch "$tag" "https://github.com/$REPO.git" "$src"
  fi

  local gobin
  gobin="$(go env GOBIN)"
  [[ -n "$gobin" ]] || gobin="$(go env GOPATH)/bin"
  export PATH="$gobin:$PATH"
  if ! have wails3; then
    info "Installation de Wails $WAILS_VERSION…"
    go install "github.com/wailsapp/wails/v3/cmd/wails3@$WAILS_VERSION"
  fi

  (cd "$src/frontend" && npm ci --no-audit --no-fund --loglevel=error)
  (cd "$src" && wails3 task linux:build VERSION="${tag#v}")
  [[ -x "$src/build/bin/$APP" ]] || die "le build n'a pas produit build/bin/$APP."

  local bin_dir="$HOME/.local/bin"
  local share="${XDG_DATA_HOME:-$HOME/.local/share}"
  mkdir -p "$bin_dir" "$share/applications" "$share/icons/hicolor/1024x1024/apps"
  install -m 0755 "$src/build/bin/$APP" "$bin_dir/verk"
  install -m 0644 "$src/build/appicon.png" "$share/icons/hicolor/1024x1024/apps/verk.png"
  cat >"$share/applications/verk.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=$APP
Comment=Tâches, sous-tâches et coach Claude Code
Exec=$bin_dir/verk
Icon=verk
Terminal=false
Categories=Office;Utility;
StartupWMClass=$APP
EOF
  have update-desktop-database && update-desktop-database "$share/applications" >/dev/null 2>&1 || true
  have gtk-update-icon-cache && gtk-update-icon-cache -q "$share/icons/hicolor" >/dev/null 2>&1 || true

  info "$APP $tag est installé : $bin_dir/verk"
  case ":$PATH:" in
    *":$bin_dir:"*) ;;
    *) echo "    Ajoute $bin_dir à ton PATH pour lancer « verk » depuis un terminal." ;;
  esac
  echo "    Il apparaît aussi dans le menu des applications."
}

case "$(uname -s)" in
  Darwin) install_macos ;;
  Linux) install_linux ;;
  *) die "système non pris en charge : $(uname -s). Sous Windows, utilise scripts/install.ps1." ;;
esac
