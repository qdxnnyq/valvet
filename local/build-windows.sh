#!/bin/sh
# Windows GUI build. -H windowsgui stops the black console window.
set -eu
cd "$(dirname "$0")"
LABEL=$(python3 -c 'import re; t=open("update.go",encoding="utf-8").read(); m=re.search(r"appLabel = \"([^\"]+)\"", t); assert m; print(m.group(1))')
restore_setup() {
  python3 - << 'PY'
import pathlib
for path in (pathlib.Path("setup/ui.html"), pathlib.Path("setup/main.go")):
    bak = pathlib.Path(str(path) + ".bak")
    if bak.exists():
        path.write_text(bak.read_text(encoding="utf-8"), encoding="utf-8")
        bak.unlink()
PY
}
trap restore_setup EXIT
python3 - "$LABEL" << 'PY'
import pathlib, sys
label = sys.argv[1]
for path in (pathlib.Path("setup/ui.html"), pathlib.Path("setup/main.go")):
    text = path.read_text(encoding="utf-8")
    pathlib.Path(str(path) + ".bak").write_text(text, encoding="utf-8")
    path.write_text(text.replace("__VALVET__", label), encoding="utf-8")
PY
export CGO_ENABLED=0
export GOOS=windows
export GOARCH=amd64
go build -trimpath -ldflags "-s -w -H windowsgui" -o /workspace/public/Valvet.exe .
cp -f /workspace/public/Valvet.exe ./setup/Valvet.exe
cp -f ./rsrc_windows_amd64.syso ./setup/rsrc_windows_amd64.syso
(
  cd setup
  go build -tags setup -trimpath -ldflags "-s -w -H windowsgui" -o /workspace/public/ValvetSetup.exe .
)
rm -f ./setup/Valvet.exe ./setup/rsrc_windows_amd64.syso
rm -f /workspace/public/Velvet.exe /workspace/public/VelvetSetup.exe /workspace/artifacts/Velvet.exe /workspace/artifacts/VelvetSetup.exe
cp -f /workspace/public/Valvet.exe /workspace/artifacts/Valvet.exe
cp -f /workspace/public/ValvetSetup.exe /workspace/artifacts/ValvetSetup.exe
python3 - "$LABEL" << 'PY'
import struct, sys
label = sys.argv[1].encode()
for p in ("/workspace/public/Valvet.exe", "/workspace/public/ValvetSetup.exe"):
    d=open(p,"rb").read(4096)
    e=struct.unpack_from("<I", d, 0x3C)[0]
    sub=struct.unpack_from("<H", d, e+24+68)[0]
    print(p, "subsystem", sub, "bytes", __import__("os").path.getsize(p))
    if sub != 2:
        raise SystemExit("expected GUI subsystem 2")
blob=open("/workspace/public/Valvet.exe","rb").read()
if label not in blob:
    raise SystemExit("missing version label")
if b"Velvet.exe" in blob and b"Valvet.exe" not in blob:
    raise SystemExit("unexpected velvet-only binary")
PY
