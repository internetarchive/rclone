#!/bin/bash

set -eu
set -o pipefail

LATEST=https://github.com/internetarchive/rclone/releases/latest
TEMPLATE=README.template

for link in $(curl -sL $LATEST | grep -Eo "/internetarchive/rclone/releases/download/[^\"]*" | grep -v "checksums" | awk '{print "https://github.com"$0}'); do
	case $link in
	*Darwin_arm64)
		v="Apple ARM"
		RELEASE_ASSET_DARWIN_ARM=$link
		;;
	*Darwin_x86_64)
		v="Apple Intel"
		RELEASE_ASSET_DARWIN_INTEL=$link
		;;
	*Linux_arm64)
		v="Linux ARM"
		;;
	*Linux_x86_64)
		v="Linux Intel"
		;;
	*Windows_arm64.exe)
		v="Windows ARM"
		;;
	*Windows_x86_64.exe)
		v="Windows Intel"
		;;
	*) ;;
	esac
	snippet+="* [$v]($link)\n"
done

sed -e "s@RELEASE_ASSET_LINKS@$snippet@" $TEMPLATE |
	sed -e "s@RELEASE_ASSET_DARWIN_ARM@$RELEASE_ASSET_DARWIN_ARM@" |
	sed -e "s@RELEASE_ASSET_DARWIN_INTEL@$RELEASE_ASSET_DARWIN_INTEL@"
