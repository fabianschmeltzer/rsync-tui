# rsync-srv-gui

`rsync-srv-gui` is a small Bash tool with a Whiptail-based menu interface for using `rsync` safely and conveniently on systems with `/srv`-based storage mounts.

The tool is especially useful for Raspberry Pi, OpenMediaVault, USB storage, and homelab setups where data needs to be copied, moved, mirrored, or tested between mounted drives, backup targets, or service directories.

## Features

* Folder selection via menu starting from `/srv`
* Copy new and updated files
* Move files and remove empty source folders afterward
* Mirror synchronization from source to destination
* Dry-run mode without making changes
* Progress display using `rsync --info=progress2`
* Safety check against identical source and destination paths
* Safety check against selecting a destination inside the source path
* Final overview and confirmation before execution

## Modes

| Mode | Description                                                                          |
| ---- | ------------------------------------------------------------------------------------ |
| Copy | Transfers new and updated files                                                      |
| Move | Transfers files and removes them from the source afterward                           |
| Sync | Mirrors the source to the destination and deletes differing files in the destination |
| Test | Performs a dry run without making changes                                            |

## Target audience

This script is intended for Linux systems where multiple drives or data directories are mounted under `/srv`, for example:

* Raspberry Pi servers
* OpenMediaVault systems
* USB HDD/SSD setups
* Backup and restore scenarios
* Docker or homelab storage structures

## Requirements

* Linux
* Bash
* rsync
* whiptail

Install dependencies on Debian, Ubuntu, or Raspberry Pi OS:

```bash
sudo apt update
sudo apt install rsync whiptail
```

## Warning

Sync mode uses `rsync --delete`. This makes the destination match the source exactly. Files that exist only in the destination may be deleted. The script therefore shows a final summary and asks for confirmation before execution.
