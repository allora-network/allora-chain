#!/bin/bash
set -eu

YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

RESTORED_FLAG="${APP_HOME}/snapshot-restored.txt"
S3_BUCKET="allora-edgenet-backups"
LATEST_BACKUP_FILE_NAME="latest_backup.txt"
RCLONE_S3_NAME="allora_s3" #! Replace with your rclone S3 name

echo -e "Please ensure ${GREEN}allorad, rclone and zstd${NC} are installed and setup on your machine before running this script."
echo -e "For rclone, please make sure to set ${GREEN}requester_pays: true${NC} in advance configuration"
echo "After installing, re-run this script."

read -p "$(echo -e ${YELLOW}'Press [Enter] key to continue if dependencies are already installed... '${NC})"

if [ ! -f "$RESTORED_FLAG" ]; then
  echo "Restoring the node from backup..."

  #* Define the latest archive and log file
  LATEST_BACKUP_FILE=$(rclone cat "$RCLONE_S3_NAME:$S3_BUCKET/$LATEST_BACKUP_FILE_NAME")
  LOGFILE="${APP_HOME}/restore.log"

  #* Download the archive from S3 and extract it to the /data directory
  mkdir -p "${APP_HOME}/data"
  touch "$LOGFILE"
  rm -rf "${APP_HOME}/data/*"
  rclone -v cat "$RCLONE_S3_NAME:$S3_BUCKET/$LATEST_BACKUP_FILE" | tar --zstd -xvf - -C "${APP_HOME}/data" > "$LOGFILE" 2>&1
  tail -n 50 "$LOGFILE"

  echo "$LATEST_BACKUP_FILE" > "$RESTORED_FLAG"

  # Get the user and group ID of the current user
  USER_ID=$(id -u)
  GROUP_ID=$(id -g)

  # Change ownership to the current user
  chown -R "$USER_ID:$GROUP_ID" "${APP_HOME}/data"
else
  RESTORED_SNAPSHOT=$(basename "$(cat "$RESTORED_FLAG")" .tar.zst)
  echo -e "Node already restored with snapshot ${GREEN}$RESTORED_SNAPSHOT${NC}"
  echo -e "${RED}To restore from the latest snapshot, remove the file: $RESTORED_FLAG${NC}"
fi
