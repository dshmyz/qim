#!/bin/sh

PACKAGE_PATH="$1"

if [ -z "$PACKAGE_PATH" ] || [ ! -f "$PACKAGE_PATH" ]; then
  echo "Error: Package file not found: $PACKAGE_PATH"
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "Error: This script must be run as root"
  exit 1
fi

detect_package_type() {
  case "$PACKAGE_PATH" in
    *.deb) echo "deb" ;;
    *.rpm) echo "rpm" ;;
    *) echo "unknown" ;;
  esac
}

install_deb() {
  echo "Installing deb package: $PACKAGE_PATH"
  dpkg -i "$PACKAGE_PATH"
  local exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo "Dependency fix: apt-get install -f -y"
    apt-get install -f -y
    exit_code=$?
  fi
  return $exit_code
}

install_rpm() {
  echo "Installing rpm package: $PACKAGE_PATH"
  rpm -Uvh "$PACKAGE_PATH"
  return $?
}

cleanup_temp() {
  rm -f "$PACKAGE_PATH"
}

PACKAGE_TYPE=$(detect_package_type)
echo "Detected package type: $PACKAGE_TYPE"

case "$PACKAGE_TYPE" in
  deb)
    install_deb
    INSTALL_EXIT_CODE=$?
    ;;
  rpm)
    install_rpm
    INSTALL_EXIT_CODE=$?
    ;;
  *)
    echo "Error: Unsupported package type. File: $PACKAGE_PATH"
    exit 1
    ;;
esac

if [ $INSTALL_EXIT_CODE -eq 0 ]; then
  echo "QIM has been updated successfully!"
  cleanup_temp
  exit 0
else
  echo "Error: Update installation failed (exit code: $INSTALL_EXIT_CODE)"
  exit $INSTALL_EXIT_CODE
fi
