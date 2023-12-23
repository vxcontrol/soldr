#!/bin/bash

#Generate application uninstallers for macOS.

#Parameters
DATE=`date +%Y-%m-%d`
TIME=`date +%H:%M:%S`
LOG_PREFIX="[$DATE $TIME]"

#Functions
log_info() {
    echo "${LOG_PREFIX}[INFO]" $1
}

log_warn() {
    echo "${LOG_PREFIX}[WARN]" $1
}

log_error() {
    echo "${LOG_PREFIX}[ERROR]" $1
}

#Check running user
if (( $EUID != 0 )); then
    echo "Please run as root."
    exit
fi

CURRENT_VERSION=$(/Library/__PRODUCT__/__PRODUCT__ -version)

echo "Welcome to Application Uninstaller"
echo "The following packages will be REMOVED:"
echo "  __PRODUCT__-${CURRENT_VERSION}"
while true; do
    read -p "Do you wish to continue [Y/n]?" answer
    [[ $answer == "y" || $answer == "Y" || $answer == "" ]] && break
    [[ $answer == "n" || $answer == "N" ]] && exit 0
    echo "Please answer with 'y' or 'n'"
done


#Need to replace these with install preparation script
VERSION=__VERSION__
PRODUCT=__PRODUCT__

echo "Application uninstalling process started"

/Library/${PRODUCT}/${PRODUCT} -command stop || true
/Library/${PRODUCT}/${PRODUCT} -command uninstall || true
#forget from pkgutil
pkgutil --forget "org.$PRODUCT.$VERSION" > /dev/null 2>&1
if [ $? -eq 0 ]
then
  echo "[1/2] [DONE] Successfully deleted application informations"
else
  echo "[1/2] [ERROR] Could not delete application informations" >&2
fi

#remove application source distribution
rm /Applications/${PRODUCT} || true
[ -e "/Library/${PRODUCT}/${PRODUCT}" ] && rm -rf "/Library/${PRODUCT}"
if [ $? -eq 0 ]
then
  echo "[2/2] [DONE] Successfully deleted application"
else
  echo "[2/2] [ERROR] Could not delete application" >&2
fi

echo "Application uninstall process finished"
exit 0
