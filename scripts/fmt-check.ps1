# Check Go source formatting on Windows.
#
# This exists because the make recipe it replaces could not work: it inlined a
# PowerShell command with a `$$`-escaped variable, and the escaping was consumed
# before PowerShell saw it, so the command arrived as `if () { ... }` and failed
# with a parser error. `make lint` and `make check` were therefore unusable on
# Windows for everyone.
#
# Reads the current directory, so the make target runs it once per module the way
# `gofmt -l .` expects.

$ErrorActionPreference = 'Stop'

$unformatted = @(gofmt -l .)
if ($unformatted.Count -eq 0) {
    exit 0
}

Write-Error ("Go source files are not formatted:`n" + ($unformatted -join "`n"))
exit 1
