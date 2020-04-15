Summary: Create (Scheduled) Encrypted Backups of Your Bitwarden Vault 
Name: portwarden
Version: 1.0.0
Release: 1%{?dist}
License: MIT
Group: Productivity/File utilities
URL: https://github.com/vwxyzjn/portwarden
Source: https://github.com/vwxyzjn/portwarden/releases/download/%{version}/portwarden_linux_amd64
ExclusiveArch: x86_64

%description
This project creates encrypted backups for Bitwarden vaults including
attachments. It pulls your vault items from Bitwarden CLI and download all the
attachments associated with those items to a temporary backup folder. Then,
portwarden zip that folder, encrypt it with a passphrase, and delete the
temporary folder.


%prep


%build


%install
install -D -p -m 0755 %{SOURCE0} %{buildroot}%{_bindir}/portwarden


%files
/usr/bin/portwarden


%changelog
* Tue Apr 14 2020 David Castañeda <edupr91@gmail.com> 1.0.0-1
- Initial RPM release.

