# Installs the example app silently, first to Program Files and then to a
# folder of its own, checks what the installer put in place each time, and
# uninstalls it again. Needs an elevated shell.
param(
    [Parameter(Mandatory)] [string] $Installer,
    [Parameter(Mandatory)] [string] $Payload
)
$ErrorActionPreference = 'Stop'

$name = 'PommelExample'
$exe = "$name.exe"
$key = "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\$name"
$task = "$name Update"
$env:POMMELPORTER_PAYLOAD = (Resolve-Path $Payload).Path
$Installer = (Resolve-Path $Installer).Path

function Run([string] $file, [string[]] $arguments) {
    $p = Start-Process -FilePath $file -ArgumentList $arguments -Wait -PassThru -NoNewWindow
    if ($p.ExitCode -ne 0) { throw "$file $arguments exited with $($p.ExitCode)" }
}

function Expect([bool] $ok, [string] $what) {
    if (-not $ok) { throw "expected $what" }
    Write-Host "ok: $what"
}

function CheckInstalled([string] $dir) {
    Expect (Test-Path "$dir\$exe") "$exe in $dir"
    Expect (Test-Path "$dir\uninstall.exe") "uninstall.exe in $dir"
    $entry = Get-ItemProperty $key
    Expect ($entry.InstallLocation -eq $dir) "InstallLocation $dir, got $($entry.InstallLocation)"
    Expect ($entry.DisplayVersion -eq '1.0.0') 'DisplayVersion 1.0.0'
    schtasks /Query /TN $task | Out-Null
    Expect ($LASTEXITCODE -eq 0) "the task $task"
    Expect (Test-Path "$env:ProgramData\Microsoft\Windows\Start Menu\Programs\$name.lnk") 'the Start menu entry'

    $acl = Get-Acl $dir
    Expect ($acl.Owner -eq 'BUILTIN\Administrators') "owner Administrators, got $($acl.Owner)"
    Expect $acl.AreAccessRulesProtected 'no inherited rights'
    $users = $acl.Access | Where-Object { $_.IdentityReference -eq 'BUILTIN\Users' }
    $write = [System.Security.AccessControl.FileSystemRights]'Write, Delete, DeleteSubdirectoriesAndFiles, ChangePermissions, TakeOwnership'
    Expect (-not ($users | Where-Object { $_.FileSystemRights -band $write })) 'no write rights for Users'
}

function Uninstall([string] $dir) {
    Run "$dir\uninstall.exe" @('--uninstall', '/S')
    Expect (-not (Test-Path $key)) 'the Apps entry removed'
    schtasks /Query /TN $task 2>$null | Out-Null
    Expect ($LASTEXITCODE -ne 0) 'the task removed'
    Expect (-not (Test-Path "$dir\$exe")) "$exe removed"
    Expect (-not (Test-Path "$env:ProgramData\Microsoft\Windows\Start Menu\Programs\$name.lnk")) 'the Start menu entry removed'
}

$default = "$env:ProgramFiles\$name"
Run $Installer @('/S')
CheckInstalled $default
Uninstall $default

$custom = "$env:SystemDrive\PommelTest\$name"
Run $Installer @('/S', "/D=$custom")
CheckInstalled $custom
Uninstall $custom

# Below a folder ordinary users may change, the update task runs as the person
# installing instead of SYSTEM, who may then change the install folder.
$open = "$env:SystemDrive\OpenForAll"
New-Item -ItemType Directory $open | Out-Null
icacls $open /grant '*S-1-5-32-545:(OI)(CI)M' | Out-Null
$exposed = "$open\$name"
Run $Installer @('/S', "/D=$exposed")
Expect (Test-Path "$exposed\$exe") "$exe in $exposed"
$me = [System.Security.Principal.WindowsIdentity]::GetCurrent().User.Value
[xml] $definition = (schtasks /Query /TN $task /XML) -join "`n"
$principal = $definition.Task.Principals.Principal
Expect ($principal.UserId -eq $me) "the task running as $me, got $($principal.UserId)"
Expect ($principal.LogonType -eq 'S4U') 'the task running whether or not that person is signed in'
# schtasks leaves out a run level that is the default, which is no elevation.
Expect ($principal.RunLevel -in @($null, '', 'LeastPrivilege')) "the task running without elevation, got $($principal.RunLevel)"
$mine = (Get-Acl $exposed).Access | Where-Object { $_.IdentityReference.Translate([System.Security.Principal.SecurityIdentifier]).Value -eq $me }
Expect ($mine | Where-Object { $_.FileSystemRights -band [System.Security.AccessControl.FileSystemRights]::Modify }) 'change rights for that person'
Uninstall $exposed
