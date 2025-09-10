#!/usr/bin/env bash
set -e
set -x  # показывать команды

# --- Проверка Rust ---
if ! command -v cargo &>/dev/null; then
    echo "Rust не найден. Автоустановка..."
    curl https://sh.rustup.rs -sSf | sh -s -- -y
    export PATH="$HOME/.cargo/bin:$PATH"
fi

# --- Определяем OS и ARCH ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

TARGET=""
EXT=""

case "$OS" in
    darwin)
        if [ "$ARCH" = "arm64" ]; then
            TARGET="aarch64-apple-darwin"
        else
            TARGET="x86_64-apple-darwin"
        fi
        EXT="dylib"
        ;;
    linux)
        if [ "$ARCH" = "aarch64" ]; then
            TARGET="aarch64-unknown-linux-gnu"
        else
            TARGET="x86_64-unknown-linux-gnu"
        fi
        EXT="so"
        ;;
    mingw*|msys*|cygwin*)
        if [ "$ARCH" = "aarch64" ]; then
            TARGET="aarch64-pc-windows-gnu"
        else
            TARGET="x86_64-pc-windows-gnu"
        fi
        EXT="dll"
        ;;
    *)
        echo "Unsupported OS/ARCH: $OS/$ARCH"
        exit 1
        ;;
esac

echo "Building for target: $TARGET"

# --- Проверка и установка target Rust ---
if ! rustup target list | grep -q "${TARGET} (installed)"; then
    rustup target add $TARGET
fi

# --- Сборка Rust динамической библиотеки ---
cargo build --release --target $TARGET --manifest-path ./Cargo.toml

# --- Копируем библиотеку в каталог с Go пакетом ---
LIB_NAME="libalchemist_c"
BUILD_DIR="target/$TARGET/release"

cp "$BUILD_DIR/$LIB_NAME.$EXT" "$LIB_NAME.$EXT"

echo "Библиотека $LIB_NAME.$EXT готова и лежит в корне проекта"
