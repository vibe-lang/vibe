#!/bin/sh
# Vibe Language Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/vibe-lang/vibe/main/install.sh | sh
#
# Environment variables:
#   VIBE_INSTALL  - Installation directory (default: $HOME/.vibe)
#   VIBE_VERSION  - Specific version to install (default: latest)

set -eu

GITHUB_REPO="vibe-lang/vibe"
BINARY_NAME="vibe"

# ---------------------------------------------------------------------------
# Terminal colors (disabled when not a TTY)
# ---------------------------------------------------------------------------

setup_colors() {
    if [ -t 1 ] && [ -t 2 ]; then
        BOLD="$(tput bold 2>/dev/null || printf '')"
        DIM="$(tput setaf 7 2>/dev/null || printf '')"
        RED="$(tput setaf 1 2>/dev/null || printf '')"
        GREEN="$(tput setaf 2 2>/dev/null || printf '')"
        YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
        BLUE="$(tput setaf 4 2>/dev/null || printf '')"
        CYAN="$(tput setaf 6 2>/dev/null || printf '')"
        RESET="$(tput sgr0 2>/dev/null || printf '')"
    else
        BOLD=""
        DIM=""
        RED=""
        GREEN=""
        YELLOW=""
        BLUE=""
        CYAN=""
        RESET=""
    fi
}

# ---------------------------------------------------------------------------
# Logging helpers
# ---------------------------------------------------------------------------

info() {
    printf '%s  %s%s\n' "${BLUE}${BOLD}" "$*" "${RESET}"
}

success() {
    printf '%s  %s%s\n' "${GREEN}" "$*" "${RESET}"
}

warn() {
    printf '%s  warning: %s%s\n' "${YELLOW}" "$*" "${RESET}" >&2
}

error() {
    printf '%s  error: %s%s\n' "${RED}" "$*" "${RESET}" >&2
}

die() {
    error "$@"
    exit 1
}

# ---------------------------------------------------------------------------
# Utility functions
# ---------------------------------------------------------------------------

has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

need_cmd() {
    if ! has_cmd "$1"; then
        die "required command '$1' not found. Please install it and try again."
    fi
}

# ---------------------------------------------------------------------------
# HTTP download abstraction (curl preferred, wget fallback)
# ---------------------------------------------------------------------------

download() {
    url="$1"
    output="$2"

    if has_cmd curl; then
        curl --proto '=https' --tlsv1.2 -fsSL "$url" -o "$output"
    elif has_cmd wget; then
        wget -q --https-only "$url" -O "$output"
    else
        die "neither 'curl' nor 'wget' found. Please install one of them and try again."
    fi
}

download_with_progress() {
    url="$1"
    output="$2"

    if has_cmd curl; then
        if [ -t 1 ]; then
            curl --proto '=https' --tlsv1.2 -fSL --progress-bar "$url" -o "$output"
        else
            curl --proto '=https' --tlsv1.2 -fsSL "$url" -o "$output"
        fi
    elif has_cmd wget; then
        if [ -t 1 ]; then
            wget --https-only "$url" -O "$output"
        else
            wget -q --https-only "$url" -O "$output"
        fi
    else
        die "neither 'curl' nor 'wget' found. Please install one of them and try again."
    fi
}

# ---------------------------------------------------------------------------
# Platform detection
# ---------------------------------------------------------------------------

detect_platform() {
    os="$(uname -s)"
    arch="$(uname -m)"

    # Normalize OS
    case "$os" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *)      die "unsupported operating system: $os. Vibe supports Linux and macOS." ;;
    esac

    # Rosetta 2 detection on macOS: uname may report x86_64 on Apple Silicon
    if [ "$os" = "darwin" ] && [ "$arch" = "x86_64" ]; then
        if sysctl -n sysctl.proc_translated 2>/dev/null | grep -q 1; then
            arch="arm64"
        fi
    fi

    # Normalize architecture
    case "$arch" in
        x86_64 | x86-64 | x64 | amd64)  arch="amd64" ;;
        aarch64 | arm64)                   arch="arm64" ;;
        *)  die "unsupported architecture: $arch. Vibe supports x86_64 (amd64) and arm64." ;;
    esac

    PLATFORM="${os}"
    ARCH="${arch}"
}

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------

resolve_version() {
    if [ -n "${VIBE_VERSION:-}" ]; then
        # Strip leading 'v' if provided
        VERSION="$(echo "$VIBE_VERSION" | sed 's/^v//')"
        return
    fi

    info "Fetching latest version..."

    tmpfile="$(mktemp)"
    if ! download "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" "$tmpfile" 2>/dev/null; then
        rm -f "$tmpfile"
        die "failed to fetch latest release from GitHub. Check your internet connection."
    fi

    # Parse tag_name from JSON without jq (POSIX-compatible)
    VERSION="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p' "$tmpfile" | head -n 1)"
    rm -f "$tmpfile"

    if [ -z "$VERSION" ]; then
        die "could not determine latest version. Try setting VIBE_VERSION manually."
    fi
}

# ---------------------------------------------------------------------------
# Checksum verification
# ---------------------------------------------------------------------------

verify_checksum() {
    archive_path="$1"
    checksums_path="$2"
    archive_name="$3"

    # Extract expected checksum for our archive
    expected="$(grep "$archive_name" "$checksums_path" | awk '{print $1}')"

    if [ -z "$expected" ]; then
        warn "checksum not found for $archive_name in checksums.txt. Skipping verification."
        return 0
    fi

    # Compute actual checksum
    if has_cmd sha256sum; then
        actual="$(sha256sum "$archive_path" | awk '{print $1}')"
    elif has_cmd shasum; then
        actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
    else
        warn "neither 'sha256sum' nor 'shasum' found. Skipping checksum verification."
        return 0
    fi

    if [ "$expected" != "$actual" ]; then
        die "checksum verification failed!
    expected: $expected
    actual:   $actual
This could indicate a corrupted download or a tampered release."
    fi
}

# ---------------------------------------------------------------------------
# Shell config detection and PATH setup
# ---------------------------------------------------------------------------

detect_shell_config() {
    shell_name="$(basename "${SHELL:-/bin/sh}")"

    case "$shell_name" in
        bash)
            if [ "$(uname -s)" = "Darwin" ]; then
                SHELL_CONFIG="$HOME/.bash_profile"
            else
                SHELL_CONFIG="$HOME/.bashrc"
            fi
            SHELL_NAME="bash"
            ;;
        zsh)
            SHELL_CONFIG="$HOME/.zshrc"
            SHELL_NAME="zsh"
            ;;
        fish)
            SHELL_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish"
            SHELL_NAME="fish"
            ;;
        *)
            SHELL_CONFIG=""
            SHELL_NAME="$shell_name"
            ;;
    esac
}

update_shell_config() {
    install_bin="$1"

    detect_shell_config

    # Check if already in PATH
    case ":${PATH}:" in
        *":${install_bin}:"*)
            return 0
            ;;
    esac

    if [ -z "$SHELL_CONFIG" ]; then
        return 0
    fi

    # Fish uses a different syntax
    if [ "$SHELL_NAME" = "fish" ]; then
        path_line="fish_add_path \"${install_bin}\""
    else
        path_line="export PATH=\"${install_bin}:\$PATH\""
    fi

    # Check if already added to config (idempotent)
    if [ -f "$SHELL_CONFIG" ] && grep -qF "$install_bin" "$SHELL_CONFIG" 2>/dev/null; then
        return 0
    fi

    # Create config file parent directory if needed (for fish)
    config_dir="$(dirname "$SHELL_CONFIG")"
    if [ ! -d "$config_dir" ]; then
        mkdir -p "$config_dir"
    fi

    printf '\n# Vibe Language\n%s\n' "$path_line" >> "$SHELL_CONFIG"
    info "Updated ${SHELL_CONFIG}"

    NEEDS_SHELL_RELOAD="yes"
}

# ---------------------------------------------------------------------------
# Main installation logic
# ---------------------------------------------------------------------------

main() {
    setup_colors

    printf '\n'
    printf '%s  Vibe Language Installer%s\n' "${BOLD}${CYAN}" "${RESET}"
    printf '\n'

    # Check for required commands
    need_cmd tar
    need_cmd uname

    # Detect platform
    detect_platform
    success "Detected platform... ${PLATFORM} ${ARCH}"

    # Resolve version
    resolve_version
    success "Version... v${VERSION}"

    # Determine install directory
    INSTALL_DIR="${VIBE_INSTALL:-$HOME/.vibe}"
    BIN_DIR="${INSTALL_DIR}/bin"

    # Construct download URLs
    archive_name="vibe_${VERSION}_${PLATFORM}_${ARCH}.tar.gz"
    base_url="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}"
    archive_url="${base_url}/${archive_name}"
    checksums_url="${base_url}/checksums.txt"

    # Create temp directory with cleanup trap
    TMP_DIR="$(mktemp -d)"
    trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

    archive_path="${TMP_DIR}/${archive_name}"
    checksums_path="${TMP_DIR}/checksums.txt"

    # Download archive
    info "Downloading vibe v${VERSION} for ${PLATFORM}/${ARCH}..."
    if ! download_with_progress "$archive_url" "$archive_path"; then
        die "failed to download ${archive_url}
Please check that version v${VERSION} exists at:
  https://github.com/${GITHUB_REPO}/releases"
    fi

    # Download and verify checksum
    info "Verifying checksum..."
    if download "$checksums_url" "$checksums_path" 2>/dev/null; then
        verify_checksum "$archive_path" "$checksums_path" "$archive_name"
        success "Checksum verified"
    else
        warn "could not download checksums.txt. Skipping verification."
    fi

    # Extract archive
    tar -xzf "$archive_path" -C "$TMP_DIR"

    # Find the binary (may be at top level or nested)
    if [ -f "${TMP_DIR}/${BINARY_NAME}" ]; then
        binary_path="${TMP_DIR}/${BINARY_NAME}"
    else
        die "binary '${BINARY_NAME}' not found in archive. The release may be corrupted."
    fi

    # Install binary
    mkdir -p "$BIN_DIR"
    cp "$binary_path" "${BIN_DIR}/${BINARY_NAME}"
    chmod +x "${BIN_DIR}/${BINARY_NAME}"

    success "Installed to ${BIN_DIR}/${BINARY_NAME}"

    # Set up PATH
    NEEDS_SHELL_RELOAD="no"
    update_shell_config "$BIN_DIR"

    # Print success message
    printf '\n'
    printf '%s  vibe v%s installed successfully!%s\n' "${GREEN}${BOLD}" "$VERSION" "${RESET}"
    printf '\n'

    if [ "$NEEDS_SHELL_RELOAD" = "yes" ]; then
        printf '  To get started, run this command to add vibe to your PATH:\n'
        printf '\n'
        if [ "$SHELL_NAME" = "fish" ]; then
            # shellcheck disable=SC2016
            printf '    %sset -gx PATH %s $PATH%s\n' "${CYAN}" "$BIN_DIR" "${RESET}"
        else
            # shellcheck disable=SC2016
            printf '    %sexport PATH="%s:$PATH"%s\n' "${CYAN}" "$BIN_DIR" "${RESET}"
        fi
        printf '\n'
        printf '  %sThis is already saved to %s for future sessions.%s\n' "${DIM}" "$SHELL_CONFIG" "${RESET}"
        printf '\n'
    fi

    printf '  Try it out:\n'
    printf '\n'
    printf '    %svibe --version%s\n' "${CYAN}" "${RESET}"
    printf '    %svibe run examples/hello.vb%s\n' "${CYAN}" "${RESET}"
    printf '\n'

    # Try to verify the installation
    if has_cmd "${BIN_DIR}/${BINARY_NAME}"; then
        installed_version="$("${BIN_DIR}/${BINARY_NAME}" --version 2>/dev/null || true)"
        if [ -n "$installed_version" ]; then
            printf '  %s%s%s\n' "${DIM}" "$installed_version" "${RESET}"
            printf '\n'
        fi
    fi
}

main "$@"
