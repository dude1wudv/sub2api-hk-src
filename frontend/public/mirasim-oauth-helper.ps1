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
$callbackListener = $null
$callbackTask = $null
$mainTask = $listener.GetContextAsync()

try {
    while ($listener.IsListening) {
        if ($null -ne $pending -and [DateTimeOffset]::UtcNow -gt $pending.ExpiresAt) {
            $callbackListener.Close()
            $callbackListener = $null
            $callbackTask = $null
            $pending = $null
        }
        $tasks = @($mainTask)
        if ($null -ne $callbackTask) { $tasks += $callbackTask }
        $completed = [System.Threading.Tasks.Task]::WaitAny([System.Threading.Tasks.Task[]]$tasks, 1000)
        if ($completed -lt 0) { continue }
        $isCallback = $completed -eq 1
        if ($isCallback) {
            $context = $callbackTask.GetAwaiter().GetResult()
            $callbackTask = $callbackListener.GetContextAsync()
        } else {
            $context = $mainTask.GetAwaiter().GetResult()
            $mainTask = $listener.GetContextAsync()
        }
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
                    if ($isCallback -or $null -ne $pending) {
                        $response.StatusCode = 409
                        break
                    }
                    $provider = $request.QueryString['provider']
                    if ($provider -ne 'github' -and $provider -ne 'google') {
                        $response.StatusCode = 400
                        break
                    }
                    # Mirasim replaces client state. Bind this flow to its own
                    # loopback port instead of accepting callbacks on shared 8788.
                    for ($attempt = 0; $attempt -lt 10; $attempt++) {
                        $portProbe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
                        $portProbe.Start()
                        $callbackPort = $portProbe.LocalEndpoint.Port
                        $portProbe.Stop()
                        $candidate = [System.Net.HttpListener]::new()
                        $candidate.Prefixes.Add("http://127.0.0.1:$callbackPort/")
                        try { $candidate.Start(); $callbackListener = $candidate; break }
                        catch { $candidate.Close() }
                    }
                    if ($null -eq $callbackListener) { throw 'Cannot allocate OAuth callback listener' }
                    $callbackTask = $callbackListener.GetContextAsync()
                    $state = [Guid]::NewGuid().ToString('N')
                    $pending = @{ Provider = $provider; ExpiresAt = [DateTimeOffset]::UtcNow.AddMinutes(10) }
                    $redirectUri = [Uri]::EscapeDataString("http://127.0.0.1:$callbackPort/callback")
                    $authUri = "https://auth.mirasim.ai/auth/oauth/$provider/login?redirect_uri=$redirectUri&state=$state"
                    $response.Redirect($authUri)
                    break
                }
                '/callback' {
                    $token = $request.QueryString['refresh_token']
                    if (-not $isCallback -or $null -eq $pending -or [DateTimeOffset]::UtcNow -gt $pending.ExpiresAt -or [string]::IsNullOrWhiteSpace($token) -or $token.Length -gt 16384) {
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
            if ($isCallback -and $null -eq $pending -and $null -ne $callbackListener) {
                $callbackListener.Close()
                $callbackListener = $null
                $callbackTask = $null
            }
        }
    }
} finally {
    if ($null -ne $callbackListener) { $callbackListener.Close() }
    $listener.Stop()
    $listener.Close()
}
