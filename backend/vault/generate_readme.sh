#!/bin/bash

set -eu
set -o pipefail

LATEST=https://github.com/internetarchive/rclone/releases/latest
TEMPLATE=README.template

for link in $(curl -sL $LATEST | grep -Eo "/internetarchive/rclone/releases/download/[^\"]*" | grep -v "checksums" | awk '{print "https://github.com"$0}'); do
	case $link in
	*Darwin_arm64)
        v="Apple ARM (M1, M2, ...)"
		RELEASE_ASSET_DARWIN_ARM=$link
		;;
	*Darwin_x86_64)
		v="Apple Intel"
		RELEASE_ASSET_DARWIN_INTEL=$link
		;;
	*Linux_arm64)
		v="Linux ARM"
		RELEASE_ASSET_LINUX_ARM=$link
		;;
	*Linux_x86_64)
		v="Linux Intel"
		RELEASE_ASSET_LINUX_INTEL=$link
		;;
	*Windows_arm64.exe)
		v="Windows ARM"
		RELEASE_ASSET_WINDOWS_ARM=$link
		;;
	*Windows_x86_64.exe)
		v="Windows Intel"
		RELEASE_ASSET_WINDOWS_INTEL=$link
		;;
	*) ;;
	esac
	snippet+="* [$v]($link)\n"
done

sed -e "s@RELEASE_ASSET_LINKS@$snippet@g" $TEMPLATE |
	sed -e "s@RELEASE_ASSET_DARWIN_ARM@$RELEASE_ASSET_DARWIN_ARM@g" |
	sed -e "s@RELEASE_ASSET_DARWIN_INTEL@$RELEASE_ASSET_DARWIN_INTEL@g" |
	sed -e "s@RELEASE_ASSET_LINUX_ARM@$RELEASE_ASSET_LINUX_ARM@g" |
	sed -e "s@RELEASE_ASSET_LINUX_INTEL@$RELEASE_ASSET_LINUX_INTEL@g" |
	sed -e "s@RELEASE_ASSET_WINDOWS_ARM@$RELEASE_ASSET_WINDOWS_ARM@g" |
	sed -e "s@RELEASE_ASSET_WINDOWS_INTEL@$RELEASE_ASSET_WINDOWS_INTEL@g"
