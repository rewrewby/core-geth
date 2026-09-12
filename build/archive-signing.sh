#!/usr/bin/env bash

# --match 'v*' restricts this to release tags. Without it the nearest tag from
# main is archive/etclabscore-2024-12, and the slash is not cosmetic: `zip`
# refuses to create the archive while `7z` silently creates a DIRECTORY of that
# name, so the win64 leg reports success while its archives sit where the
# upload glob cannot match them. --always keeps off-tag builds working, which is
# what a workflow_dispatch rehearsal runs.
ARCHIVE_VERSION=$(git describe --tags --always --match 'v*' 2>/dev/null)

# Both OS paths must fail the same way. A name this script cannot express is a
# failed build, never a quiet half-publish.
case "$ARCHIVE_VERSION" in
  '' | */*)
    echo "archive-signing: refusing to build an archive named '${ARCHIVE_VERSION}'" >&2
    echo "  a name that is empty or contains '/' publishes nothing on win64 while passing" >&2
    exit 1
    ;;
esac

GETH_ARCHIVE_NAME="core-geth-${BUILD_OS_NAME}-${ARCHIVE_VERSION}"
ALLTOOLS_ARCHIVE_NAME="core-geth-alltools-${BUILD_OS_NAME}-${ARCHIVE_VERSION}"

if [[ "${BUILD_OS_NAME}" == "win64" ]]; then
  7z a "$GETH_ARCHIVE_NAME.zip" ./build/bin/geth.exe

  sha256sum $GETH_ARCHIVE_NAME.zip
  sha256sum $GETH_ARCHIVE_NAME.zip > $GETH_ARCHIVE_NAME.zip.sha256

  7z a "$ALLTOOLS_ARCHIVE_NAME.zip" ./build/bin/*

  sha256sum $ALLTOOLS_ARCHIVE_NAME.zip
  sha256sum $ALLTOOLS_ARCHIVE_NAME.zip > $ALLTOOLS_ARCHIVE_NAME.zip.sha256

else
  zip -j "$GETH_ARCHIVE_NAME.zip" build/bin/geth

  shasum -a 256 $GETH_ARCHIVE_NAME.zip
  shasum -a 256 $GETH_ARCHIVE_NAME.zip > $GETH_ARCHIVE_NAME.zip.sha256

  zip -j "$ALLTOOLS_ARCHIVE_NAME.zip" build/bin/*

  shasum -a 256 $ALLTOOLS_ARCHIVE_NAME.zip
  shasum -a 256 $ALLTOOLS_ARCHIVE_NAME.zip > $ALLTOOLS_ARCHIVE_NAME.zip.sha256
fi
