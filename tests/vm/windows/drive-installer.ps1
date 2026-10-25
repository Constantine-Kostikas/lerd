# Clicks through Install-Lerd.exe, or with -Uninstall its uninstaller, in the
# signed-in desktop session for run.sh. Inno's controls show to UI Automation as
# plain panes, so they are found by class and clicked with window messages.
param(
	[string] $Setup = '',
	[string] $Uninstall = '',
	[switch] $DeleteDistro,
	[Parameter(Mandatory)] [string] $Log,
	[string] $Local = '',
	[string] $Password = 'lerd-vm-test',
	[int] $TimeoutMinutes = 45
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
Add-Type -Namespace W -Name U -MemberDefinition @'
[DllImport("user32.dll", CharSet = CharSet.Unicode)]
public static extern IntPtr SendMessage(IntPtr h, uint m, IntPtr w, string l);
[DllImport("user32.dll")]
public static extern IntPtr PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
'@
$AE = [System.Windows.Automation.AutomationElement]
$TS = [System.Windows.Automation.TreeScope]
$BM_CLICK = 0x00F5
$WM_SETTEXT = 0x000C

# The installer's own process and its wizard (.tmp), under either name it has shipped as.
# The uninstaller runs as a copy of itself under %TEMP%, _unins.tmp (older Inno: _iu*).
function SetupProcesses {
	$names = if ($Uninstall) { '*unins*', '_iu*' } else { 'Install-Lerd*', 'lerd-setup*' }
	@(Get-Process -Name $names -ErrorAction SilentlyContinue)
}

function Say([string] $m) { Add-Content -Path $Log -Value ("{0:HH:mm:ss} {1}" -f (Get-Date), $m) }

function Controls($win, [string] $class) {
	@($win.FindAll($TS::Descendants, [System.Windows.Automation.Condition]::TrueCondition) |
		Where-Object { -not $_.Current.IsOffscreen -and $_.Current.ClassName -match $class })
}

function Named($win, [string] $class, [string[]] $names) {
	Controls $win $class | Where-Object { $_.Current.IsEnabled -and ($names -contains ($_.Current.Name -replace '&', '')) } |
		Select-Object -First 1
}

# Posted rather than sent: a button that opens a modal dialog would otherwise
# block this script until the dialog closed.
function Click($el) { [void][W.U]::PostMessage([IntPtr]$el.Current.NativeWindowHandle, $BM_CLICK, [IntPtr]::Zero, [IntPtr]::Zero) }

function SetText($el, [string] $s) { [void][W.U]::SendMessage([IntPtr]$el.Current.NativeWindowHandle, $WM_SETTEXT, [IntPtr]::Zero, $s) }

# The setup wizard and its message boxes, and Windows' certificate prompt,
# which belongs to another process.
function Windows {
	$ids = @(SetupProcesses | ForEach-Object Id)
	$all = $AE::RootElement.FindAll($TS::Children, [System.Windows.Automation.Condition]::TrueCondition)
	@($all | Where-Object { ($ids -contains $_.Current.ProcessId) -or ($_.Current.Name -in 'Security Warning', 'Root Certificate Store') })
}


if ($Uninstall) {
	# The uninstaller's own log, beside this one, says what it could not remove.
	$unLog = Join-Path (Split-Path $Log) 'uninstall.log'
	Say "start $Uninstall"
	Start-Process -FilePath $Uninstall -ArgumentList "/LOG=`"$unLog`"" | Out-Null
	$deadline = (Get-Date).AddMinutes($TimeoutMinutes)
	$choice = if ($DeleteDistro) { '^Delete' } else { '^Keep the distro' }
	while ((Get-Date) -lt $deadline) {
		Start-Sleep -Milliseconds 800
		if (-not (SetupProcesses)) { Say "RESULT uninstalled"; exit 0 }
		foreach ($w in Windows) {
			try {
				# The keep-or-delete question is a task dialog of command links.
				$link = Controls $w 'CCCommandLink' | Where-Object { $_.Current.Name -match $choice } | Select-Object -First 1
				if ($link) { Say "task dialog: $($link.Current.Name)"; Click $link; continue }
				$b = Named $w '^(Button|CCPushButton)$' @('Yes', 'OK')
				if ($b) {
					$text = (Controls $w 'Static' | ForEach-Object { $_.Current.Name } | Where-Object { $_ }) -join ' '
					Say "$($w.Current.Name): $($b.Current.Name -replace '&', '') ($text)"
					Click $b
				}
			} catch {
				Say "retry: $($_.Exception.Message)"
			}
		}
	}
	Say "RESULT timed out after $TimeoutMinutes minutes"
	exit 1
}

$argList = @()
if ($Local) { $argList += "/local=$Local" }
$proc = SetupProcesses | Select-Object -First 1
if ($proc) {
	Say "attach to running setup $($proc.Id)"
} else {
	Say "start $Setup $argList"
	# Start-Process refuses an empty -ArgumentList, which a published build without /local= has.
	$extra = if ($argList) { @{ ArgumentList = $argList } } else { @{} }
	$proc = Start-Process -FilePath $Setup -PassThru @extra
}
$deadline = (Get-Date).AddMinutes($TimeoutMinutes)
$filledUser = $false
$checkedDetails = $false
$lastPage = ''
while ((Get-Date) -lt $deadline) {
	Start-Sleep -Milliseconds 800
	if (-not (SetupProcesses)) {
		Say "RESULT setup exited"
		exit 0
	}
	foreach ($w in Windows) {
		try {
			if ($w.Current.Name -eq 'Security Warning') {
				$b = Named $w 'Button' @('Yes')
				if ($b) { Say "certificate prompt: Yes"; Click $b }
				continue
			}
			# A message box: standard Win32 buttons rather than Inno's own.
			$ok = Named $w '^Button$' @('OK')
			if ($ok) {
				$text = (Controls $w 'Static' | ForEach-Object { $_.Current.Name } | Where-Object { $_ }) -join ' '
				Say "message: $text"
				if ($text -match 'fail|error|could not|cannot') { Say "RESULT error dialog: $text"; exit 1 }
				Click $ok
				continue
			}
			if ($w.Current.ClassName -ne 'TWizardForm') { continue }
			$page = (Controls $w 'TNewStaticText' | Select-Object -First 1).Current.Name
			if ($page -ne $lastPage) { Say "page: $page"; $lastPage = $page }
			# The log under the progress bar is opt in; it has to open while the
			# steps run, which is when anyone wants it.
			if ($page -eq 'Installing' -and -not $checkedDetails) {
				$show = Named $w 'TNewButton' @('Show details')
				if ($show) {
					Click $show
					Start-Sleep -Seconds 3
					$memo = Controls $w 'Memo'
					if ($memo.Count -eq 0) { Say "RESULT FAIL Show details did not open the log"; exit 1 }
					Say "details: opened"
					$checkedDetails = $true
				}
				continue
			}
			$accept = Named $w 'TNewRadioButton' @('I accept the agreement')
			$next = Named $w 'TNewButton' @('Next', 'Install', 'Finish')
			if ($accept -and -not $next) { Say "license: accept"; Click $accept; continue }
			$edits = Controls $w 'Edit'
			if ($edits.Count -eq 3 -and -not $filledUser) {
				Say "user page: $($edits[0].Current.Name) / password typed twice"
				SetText $edits[1] $Password
				SetText $edits[2] $Password
				$filledUser = $true
				continue
			}
			if ($next) {
				$label = $next.Current.Name -replace '&', ''
				Say "  -> $label"
				Click $next
				if ($label -eq 'Finish') { Say "RESULT finished"; exit 0 }
				Start-Sleep -Seconds 2
			}
		} catch {
			# The window changing between finding and clicking is normal; the next
			# pass sees the new state. The message is logged in case it is not.
			Say "retry: $($_.Exception.Message)"
		}
	}
}
Say "RESULT timed out after $TimeoutMinutes minutes"
exit 1
