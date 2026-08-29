#!/usr/bin/env bash
set -e

# ==============================================================================
#  ⚡ ARIS Installer — Autonomous Reasoner for Image System
# ==============================================================================

REPO="SalvucciFacundo/ARIS"
GITHUB_URL="https://github.com/${REPO}"

# ANSI color codes
CYAN='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${CYAN}${BOLD}"
echo "  ⚡ ===================================================== ⚡"
echo "      ARIS — Autonomous Reasoner for Image System"
echo "  ⚡ ===================================================== ⚡"
echo -e "${NC}"

# 1. Detect Operating System
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  linux*)  OS="linux" ;;
  darwin*) OS="darwin" ;;
  *)
    echo -e "${RED}❌ Unsupported operating system: ${OS}${NC}"
    echo "ARIS one-line installer supports Linux and macOS. For Windows, download the zip release."
    exit 1
    ;;
esac

# 2. Detect Machine Architecture
ARCH="$(uname -m | tr '[:upper:]' '[:lower:]')"
case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo -e "${RED}❌ Unsupported architecture: ${ARCH}${NC}"
    exit 1
    ;;
esac

echo -e "🖥️  Detected Platform: ${GREEN}${OS}/${ARCH}${NC}"

# 3. Determine Installation Directory
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -n "${SUDO_USER}" ] && [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "${INSTALL_DIR}"
fi

# 4. Fetch Latest Version Tag from GitHub
echo -e "🔍 Checking latest version from GitHub..."
LATEST_TAG=""

if command -v curl >/dev/null 2>&1; then
  LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
elif command -v wget >/dev/null 2>&1; then
  LATEST_TAG=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
fi

# Fallback if no release found or rate-limited
if [ -z "${LATEST_TAG}" ]; then
  LATEST_TAG="v1.0.0"
  echo -e "${YELLOW}⚠️  Could not fetch release via GitHub API, using fallback ${LATEST_TAG}${NC}"
else
  echo -e "📦 Latest Version: ${GREEN}${LATEST_TAG}${NC}"
fi

# 5. Download and Extract Archive
TARBALL="aris_${LATEST_TAG}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="${GITHUB_URL}/releases/download/${LATEST_TAG}/${TARBALL}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

echo -e "⬇️  Downloading ${CYAN}${TARBALL}${NC}..."
if command -v curl >/dev/null 2>&1; then
  if ! curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}"; then
    DOWNLOAD_FAILED=true
  fi
elif command -v wget >/dev/null 2>&1; then
  if ! wget -q "${DOWNLOAD_URL}" -O "${TMP_DIR}/${TARBALL}"; then
    DOWNLOAD_FAILED=true
  fi
fi

if [ "${DOWNLOAD_FAILED}" = "true" ] || [ ! -f "${TMP_DIR}/${TARBALL}" ]; then
  echo -e "${YELLOW}⚠️  Pre-built release not found for ${LATEST_TAG}. Attempting install via 'go install'...${NC}"
  if command -v go >/dev/null 2>&1; then
    echo -e "🚀 Compiling from source via Go..."
    go install "github.com/${REPO}/cmd/aris@latest"
    echo -e "${GREEN}✅ Successfully installed via 'go install'!${NC}"
    exit 0
  else
    echo -e "${RED}❌ Download failed and Go is not installed.${NC}"
    echo "Please download the binary manually from: ${GITHUB_URL}/releases"
    exit 1
  fi
fi

echo -e "📦 Extracting binary..."
tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

if [ ! -f "${TMP_DIR}/aris" ]; then
  echo -e "${RED}❌ Extraction failed: binary 'aris' not found in archive.${NC}"
  exit 1
fi

echo -e "🚀 Installing binary to ${GREEN}${INSTALL_DIR}/aris${NC}..."
if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP_DIR}/aris" "${INSTALL_DIR}/aris"
  chmod +x "${INSTALL_DIR}/aris"
else
  sudo mv "${TMP_DIR}/aris" "${INSTALL_DIR}/aris"
  sudo chmod +x "${INSTALL_DIR}/aris"
fi

echo -e "\n${GREEN}${BOLD}✨ ARIS successfully installed!${NC}\n"

# Verify PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo -e "${YELLOW}⚠️  Notice: ${INSTALL_DIR} is not currently in your \$PATH.${NC}"
    echo "Add it to your shell configuration (~/.bashrc or ~/.zshrc):"
    echo -e "  ${CYAN}export PATH=\"${INSTALL_DIR}:\$PATH\"${NC}\n"
    ;;
esac

echo -e "Run ${CYAN}aris --help${NC} or ${CYAN}aris chat${NC} to start creating!"
