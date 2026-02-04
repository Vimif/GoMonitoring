# Verification Script for Go Monitoring User Management

$baseUrl = "http://localhost:8080"
$jar = New-Object Microsoft.PowerShell.Commands.WebRequestSession

Write-Host "1. Logging in as Admin..."
$loginBody = @{
    username = "admin"
    password = "admin"
}
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/login" -Method Post -Body $loginBody -SessionVariable session -ErrorAction Stop -UseBasicParsing
    $adminCookie = $session.Cookies.GetCookies($baseUrl)
    Write-Host "   Success. Cookie acquired."
}
catch {
    Write-Host "   Failed to login as admin: $_"
    exit 1
}

Write-Host "`n2. Listing Users (Admin)..."
try {
    $users = Invoke-RestMethod -Uri "$baseUrl/api/users" -Method Get -WebSession $session
    Write-Host "   Success. Found $($users.Count) users."
}
catch {
    Write-Host "   Failed to list users: $_"
    exit 1
}

Write-Host "`n3. Verifying /users Page Access (Admin)..."
try {
    $page = Invoke-WebRequest -Uri "$baseUrl/users" -Method Get -WebSession $session -UseBasicParsing
    if ($page.StatusCode -eq 200) {
        Write-Host "   Success. /users page accessible."
    }
    else {
        Write-Host "   Failed. Status check: $($page.StatusCode)"
        exit 1
    }
}
catch {
    Write-Host "   Failed to access /users page: $_"
    exit 1
}

Write-Host "`n4. Creating User 'test'..."
$newUser = @{
    username = "test"
    password = "test"
    role     = "user"
} | ConvertTo-Json

try {
    # Check if user exists first to avoid conflict error breaking script
    try {
        Invoke-RestMethod -Uri "$baseUrl/api/users" -Method Post -Body $newUser -ContentType "application/json" -WebSession $session -ErrorAction Stop
        Write-Host "   Success. User 'test' created."
    }
    catch {
        if ($_.Exception.Response.StatusCode -eq 409) {
            Write-Host "   User 'test' already exists. Continuing."
        }
        else {
            throw $_
        }
    }
}
catch {
    Write-Host "   Failed to create user: $_"
    exit 1
}

Write-Host "`n5. Logging in as 'test'..."
$testSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$testLogin = @{
    username = "test"
    password = "test"
}
try {
    Invoke-WebRequest -Uri "$baseUrl/login" -Method Post -Body $testLogin -SessionVariable testSession -ErrorAction Stop -UseBasicParsing
    Write-Host "   Success. Login as 'test' successful."
}
catch {
    Write-Host "   Failed to login as 'test': $_"
    exit 1
}

Write-Host "`n6. Checking Access to /users as 'test' (Should be Denied)..."
try {
    # -MaximumRedirection 0 to detect redirect
    $resp = Invoke-WebRequest -Uri "$baseUrl/users" -Method Get -WebSession $testSession -MaximumRedirection 0 -ErrorAction SilentlyContinue -UseBasicParsing
    
    if ($resp.StatusCode -eq 303 -or $resp.StatusCode -eq 403) {
        Write-Host "   Success. Access denied/Redirected ($($resp.StatusCode))."
    }
    elseif ($resp.StatusCode -eq 200) {
        Write-Host "   FAILURE: 'test' user got 200 OK on /users!"
        exit 1
    }
    else {
        Write-Host "   Unexpected Status: $($resp.StatusCode)"
        exit 1
    }
}
catch {
    # Some PS versions throw on 303 with MaxRedir 0
    $ex = $_.Exception
    if ($ex.Response.StatusCode -eq "SeeOther" -or $ex.Response.StatusCode -eq "Forbidden" -or $ex.Response.StatusCode -eq "Found") {
        Write-Host "   Success. Access denied/Redirected ($($ex.Response.StatusCode)) [Caught Exception]."
    }
    else {
        Write-Host "   Unexpected error: $_"
        exit 1
    }
}

Write-Host "`n7. Deleting User 'test' (Admin)..."
try {
    Invoke-RestMethod -Uri "$baseUrl/api/users/test" -Method Delete -WebSession $session
    Write-Host "   Success. User 'test' deleted."
}
catch {
    Write-Host "   Failed to delete user: $_"
    exit 1
}

Write-Host "`nVerification Complete. All checks passed."
