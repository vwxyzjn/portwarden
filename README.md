# PortWarden

This project provides encrypted backups for (Bitwarden)[https://bitwarden.com/] vaults, including attachments. It pulls your vault items from (Bitwarden CLI)[https://github.com/bitwarden/cli] and download all the attachments associated with those items to a temporary backup folder. Then, portwarden zip that folder, encrypt it with a passphrase, and delete the temporary folder. 


It addresses this issue in the community forum https://community.bitwarden.com/t/encrypted-export/235, but hopefully Bitwarden will come up with official solutions soon.

## Usage

Go to https://github.com/bitwarden/cli/releases to download the latest version of Bitwarden CLI and place the executable `bw`/`bw.exe` in your `PATH`. Then, go to https://github.com/vwxyzjn/portwarden/releases/ to downlaod the latest release of `portwarden`. Now just follow the steps in the following Gif:

![alt text](./demo.gif "Logo Title Text 1")

## Contribution 

PRs are welcome. For ideas, you could probably add a progress bar. 