param(
    [parameter(mandatory = $true)]
    [String]
    $ConfigFile = "packages.json",

    [parameter(mandatory = $true)]
    [String]
    $PackageName = "vxagent"
)

function New-AgentMSI
{
    param(
        [parameter(mandatory = $true)]
        [String]
        $WIXFile,

        [parameter(mandatory = $true)]
        [String]
        $PathToAgentExe,

        [parameter(mandatory = $true)]
        [String]
        $Arch,

        [parameter(mandatory = $true)]
        [String]
        $PackageVersion,

        [String]
        $Name,

        [PSCustomObject]
        $Parameters,

        [String]
        $OutDir
    )

    $candleArgs = @(
        "-sw1150", # suppress warning about the ServiceConfig field in the vxagent.wsx file
        "-arch", "$Arch",
        "-ext", "WixUtilExtension",
        "-dAgentSourceExecutable=$PathToAgentExe",
        "-dProductVersion=$PackageVersion"
    )

    $FileName = "{0}-{1}_{2}" -f $PackageName, $PackageVersion, $Arch
    $FileName = Join-Path $OutDir $FileName
    $candleArgs += "-o", "$FileName.wsxobj"
    $candleArgs += "$WIXFile"

    foreach ($Parameter in $Parameters.PSObject.Properties)
    {
        $candleArgs += "-d$($Parameter.Name)=$($Parameter.Value)"
    }

    $OutputFileName = "$FileName.msi"
    $lightArgs = @(
        "-sw1076",
        "-ext", "WixUtilExtension",
        "-o", $OutputFileName,
        "$FileName.wsxobj"
    )

    Write-Host "Compiling..."
    "candle.exe $candleArgs"
    & candle.exe $candleArgs
    if ($LASTEXITCODE -ne 0) { throw }

    Write-Host "Linking..."
    "light.exe $lightArgs"
    & light.exe $lightArgs
    if ($LASTEXITCODE -ne 0) { throw }

    Write-Host "MSI File: $OutputFileName"
}

#-------------------------------------------------------------------------------

$version = [IO.File]::ReadAllText("_tmp/version")
$PackageVersion = (($version -Split "v")[1] -Split "-")[0]
$Config = Get-Content $ConfigFile -Encoding UTF8 | ConvertFrom-Json

if ([String]::IsNullOrEmpty($env:WIX))
{
    Write-Warning "WIX env var is not set"
}
else
{
    Write-Host "WIX: $env:WIX"
}

#-------------------------------------------------------------------------------

Write-Host "=========== Building VXAgent MSI ==========="
foreach ($AgentArch in $Config.AgentArch.PSObject.Properties)
{
    $MSIConfig =
    @{
        Arch = $AgentArch.Name
        PathToAgentExe = $AgentArch.Value
        WIXFile = $Config.WIXFile
        OutDir = $Config.OutDir
        PackageVersion = $PackageVersion
    }

    Write-Host "=========== Build for $($MSIConfig.Arch)"
    New-AgentMSI @MSIConfig
    Write-Host "==============================================================================="
}
