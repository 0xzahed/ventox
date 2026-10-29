package writer

// Windows11BypassOptions specifies which requirements to bypass in the generated autounattend.xml.
type Windows11BypassOptions struct {
	BypassTPM        bool
	BypassSecureBoot bool
	BypassRAM        bool
	BypassCPU        bool
	BypassStorage    bool
	BypassNRO        bool // Bypass Microsoft Account requirement (Network Requirement Bypass)
}

// DefaultWindows11Bypass returns full bypass options enabled.
func DefaultWindows11Bypass() Windows11BypassOptions {
	return Windows11BypassOptions{
		BypassTPM:        true,
		BypassSecureBoot: true,
		BypassRAM:        true,
		BypassCPU:        true,
		BypassStorage:    true,
		BypassNRO:        true,
	}
}

// GenerateAutoUnattendXML creates a standards-compliant unattend configuration
// that applies LabConfig registry entries during the Windows PE setup phase.
func GenerateAutoUnattendXML(opts Windows11BypassOptions) string {
	var commands []string
	order := 1

	addCommand := func(path string, desc string) {
		commands = append(commands, `            <RunSynchronousCommand wcm:action="add">`+"\n"+
			`                <Order>`+string(rune('0'+order))+`</Order>`+"\n"+
			`                <Description>`+desc+`</Description>`+"\n"+
			`                <Path>`+path+`</Path>`+"\n"+
			`            </RunSynchronousCommand>`)
		order++
	}

	if opts.BypassTPM {
		addCommand(`reg add HKLM\SYSTEM\Setup\LabConfig /v BypassTPMCheck /t REG_DWORD /d 1 /f`, "Bypass TPM 2.0 Check")
	}
	if opts.BypassSecureBoot {
		addCommand(`reg add HKLM\SYSTEM\Setup\LabConfig /v BypassSecureBootCheck /t REG_DWORD /d 1 /f`, "Bypass Secure Boot Check")
	}
	if opts.BypassRAM {
		addCommand(`reg add HKLM\SYSTEM\Setup\LabConfig /v BypassRAMCheck /t REG_DWORD /d 1 /f`, "Bypass RAM Check")
	}
	if opts.BypassCPU {
		addCommand(`reg add HKLM\SYSTEM\Setup\LabConfig /v BypassCPUCheck /t REG_DWORD /d 1 /f`, "Bypass CPU Check")
	}
	if opts.BypassStorage {
		addCommand(`reg add HKLM\SYSTEM\Setup\LabConfig /v BypassStorageCheck /t REG_DWORD /d 1 /f`, "Bypass Storage Check")
	}
	if opts.BypassNRO {
		addCommand(`reg add HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\OOBE /v BypassNRO /t REG_DWORD /d 1 /f`, "Bypass Microsoft Online Account Requirement")
	}

	cmdXML := ""
	for _, c := range commands {
		cmdXML += c + "\n"
	}

	return `<?xml version="1.0" encoding="utf-8"?>
<unattend xmlns="urn:schemas-microsoft-com:unattend">
    <settings pass="windowsPE">
        <component name="Microsoft-Windows-Setup" processorArchitecture="amd64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State">
            <RunSynchronous>
` + cmdXML + `            </RunSynchronous>
        </component>
        <component name="Microsoft-Windows-Setup" processorArchitecture="arm64" publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State">
            <RunSynchronous>
` + cmdXML + `            </RunSynchronous>
        </component>
    </settings>
</unattend>
`
}
