$ErrorActionPreference = "Continue"
$baseUrl = "http://localhost:8080/api/v1"

function Get-Token($username, $password, $tenant) {
    if ($tenant) {
        $body = @{ tenant = $tenant; username = $username; password = $password } | ConvertTo-Json
    } else {
        $body = @{ username = $username; password = $password } | ConvertTo-Json
    }
    $res = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $body -ContentType "application/json"
    return $res.data.access_token
}

function Test-Role($role, $user, $pass, $tenantSlug="", $headerSlug="") {
    Write-Host "=== Testing $role ($user) ==="
    try {
        $token = Get-Token $user $pass $tenantSlug
        Write-Host "Login SUCCESS. Token: $($token.Substring(0, 20))..."
        
        $headers = @{ "Authorization" = "Bearer $token" }
        if ($headerSlug) { $headers["X-Tenant-Slug"] = $headerSlug }

        # Get Users
        $usersRes = Invoke-RestMethod -Uri "$baseUrl/users" -Method Get -Headers $headers
        Write-Host "GET /users SUCCESS. Found $($usersRes.data.Count) users."
        
        if ($usersRes.data.Count -gt 0) {
            $firstUser = $usersRes.data[0]
            $firstUserId = $firstUser.id
            
            # Get User by ID
            $userRes = Invoke-RestMethod -Uri "$baseUrl/users/$firstUserId" -Method Get -Headers $headers
            Write-Host "GET /users/$firstUserId SUCCESS. User role: $($userRes.data.role), Username: $($userRes.data.username)"
        }
    } catch {
        Write-Host "ERROR: $_"
        if ($_.ErrorDetails) { Write-Host "Details: $($_.ErrorDetails.Message)" }
    }
    Write-Host ""
}

# Clear redis to avoid rate limits
docker exec docker-redis-1 redis-cli FLUSHALL | Out-Null

Test-Role "Superadmin (to Alpha)" "superadmin" "superadminpass" "" "alpha"
Test-Role "Superadmin (to Beta)" "superadmin" "superadminpass" "" "beta"

Test-Role "Admin Alpha" "admin.alpha" "adminpass" "alpha"
Test-Role "Staff Alpha" "staff.alpha" "staffpass" "alpha"
Test-Role "Admin Beta" "admin.beta" "adminpass" "beta"
Test-Role "Staff Beta" "staff.beta" "staffpass" "beta"

Write-Host "=== Testing Cross-Tenant: Admin Alpha reading Beta ==="
try {
    $token = Get-Token "admin.alpha" "adminpass" "alpha"
    $headers = @{ "Authorization" = "Bearer $token"; "X-Tenant-Slug" = "beta" }
    $res = Invoke-RestMethod -Uri "$baseUrl/users" -Method Get -Headers $headers
    Write-Host "FAIL: Allowed access! Found $($res.data.Count) users."
} catch {
    Write-Host "SUCCESS: Blocked cross-tenant access. ERROR: $_"
    if ($_.ErrorDetails) { Write-Host "Details: $($_.ErrorDetails.Message)" }
}
Write-Host ""
