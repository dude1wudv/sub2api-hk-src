param(
    [string]$ReturnOrigin = 'https://sub.sunmmyapi.xyz'
)

$ErrorActionPreference = 'Stop'
$origin = [Uri]$ReturnOrigin
if ($origin.Scheme -ne 'https' -or $origin.Host -ne 'sub.sunmmyapi.xyz' -or $origin.AbsolutePath -ne '/' -or $origin.Query -ne '') {
    throw 'ReturnOrigin must be https://sub.sunmmyapi.xyz'
}

$listener = [System.Net.HttpListener]::new()
$listener.Prefixes.Add('http://127.0.0.1:8788/')
$listener.Start()
Write-Host 'Mirasim OAuth helper is listening on 127.0.0.1:8788. Keep this window open while authorizing.'
$pending = $null

try {
    while ($listener.IsListening) {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response
        $response.Headers['Cache-Control'] = 'no-store'
        $response.Headers['Referrer-Policy'] = 'no-referrer'
        try {
            if ($request.HttpMethod -ne 'GET') {
                $response.StatusCode = 405
                continue
            }
            switch ($request.Url.AbsolutePath) {
                '/start' {
                    $provider = $request.QueryString['provider']
                    if ($provider -ne 'github' -and $provider -ne 'google') {
                        $response.StatusCode = 400
                        break
                    }
                    $random = New-Object byte[] 16
                    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
                    try { $rng.GetBytes($random) } finally { $rng.Dispose() }
                    $state = [Convert]::ToBase64String($random).TrimEnd('=').Replace('+', '-').Replace('/', '_')
                    $pending = @{ Provider = $provider; ExpiresAt = [DateTimeOffset]::UtcNow.AddMinutes(10) }
                    $redirectUri = [Uri]::EscapeDataString('http://127.0.0.1:8788/callback')
                    $authUri = "https://auth.mirasim.ai/auth/oauth/$provider/login?redirect_uri=$redirectUri&state=$state"
                    $response.Redirect($authUri)
                    break
                }
                '/callback' {
                    $token = $request.QueryString['refresh_token']
                    if ($null -eq $pending -or [DateTimeOffset]::UtcNow -gt $pending.ExpiresAt -or [string]::IsNullOrWhiteSpace($token) -or $token.Length -gt 16384) {
                        $response.StatusCode = 400
                        break
                    }
                    $provider = $pending.Provider
                    $pending = $null
                    $encoded = [Uri]::EscapeDataString($token)
                    $response.Redirect("$ReturnOrigin/admin/accounts#mirasim_provider=$provider&refresh_token=$encoded")
                    break
                }
                default { $response.StatusCode = 404 }
            }
        } finally {
            $response.Close()
        }
    }
} finally {
    $listener.Stop()
    $listener.Close()
}
