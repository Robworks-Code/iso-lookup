# Installing `iso`

`iso` is a single self-contained binary. Pick whichever method suits you.

## Homebrew (macOS)

```bash
brew install robworks-code/tap/iso
```

Or tap once, then install by short name and pick up future upgrades with
`brew upgrade`:

```bash
brew tap robworks-code/tap
brew install iso
```

## Go (any platform)

Requires Go 1.25 or newer. Installs into `$(go env GOPATH)/bin` (make sure that's
on your `PATH`):

```bash
go install github.com/Robworks-Code/iso-lookup/cmd/iso@latest
```

## Prebuilt binaries

Download the tarball for your OS/architecture (darwin/linux × amd64/arm64) from
the [releases page](https://github.com/Robworks-Code/iso-lookup/releases),
extract it, and put `iso` somewhere on your `PATH`:

```bash
tar xzf iso_<version>_<os>_<arch>.tar.gz
sudo mv iso /usr/local/bin/
```

On macOS, prebuilt binaries are notarized via the Homebrew cask; if you download
a tarball directly you may need to clear the quarantine attribute:

```bash
xattr -d com.apple.quarantine ./iso
```

## From source

```bash
git clone https://github.com/Robworks-Code/iso-lookup.git
cd iso-lookup

make install            # build and install into /usr/local/bin (override PREFIX=...)
# or
make build              # build ./bin/iso only
# or
go install ./cmd/iso    # install into $(go env GOPATH)/bin
```

## Verify

```bash
iso --version
```

## First run

Download the offline metadata index before your first search (one time, then
whenever you want fresh metadata):

```bash
iso update
```

Optionally point `iso` at a folder of your own standards documents so it can read
full text:

```bash
iso config set-docs ~/standards
```

See the [README](README.md) for usage and the
[CHANGELOG](CHANGELOG.md) for release notes.
