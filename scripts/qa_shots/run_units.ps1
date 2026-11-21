$ErrorActionPreference = 'Continue'
$pkgs = @('cms','rbac','deliver','officex','malware','storage','safehtml','diff','plan','verify','corp')
foreach ($p in $pkgs) {
  go test -c -o "C:\Windows\Temp\notU_$p.exe" "./internal/$p/"
  if ($LASTEXITCODE -ne 0) { echo "$p COMPILE_FAIL"; continue }
  & "C:\Windows\Temp\notU_$p.exe" | Out-Null
  echo "$p EXIT:$LASTEXITCODE"
}
