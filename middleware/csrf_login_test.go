package middleware_test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"testing"

	"go-monitoring/auth"
	"go-monitoring/config"
	"go-monitoring/middleware"
	"go-monitoring/storage"
)

func TestLoginCSRFProtection(t *testing.T) {
	// Change working directory to project root to allow template loading
	// Assuming test is run from ./middleware directory
	if err := os.Chdir(".."); err != nil {
		// If running from root (e.g. go test ./middleware/csrf_login_test.go), this might fail or go to parent.
		// Better check if templates exists here.
		if _, err := os.Stat("templates"); os.IsNotExist(err) {
             t.Fatalf("Templates directory not found and failed to chdir: %v", err)
        }
	} else {
        // If we successfully moved up, check if templates exists
        if _, err := os.Stat("templates"); os.IsNotExist(err) {
            // Maybe we were already at root and moved up?
            os.Chdir("middleware") // go back
        }
    }

	// Setup DB
	db, err := storage.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Setup Auth
	um := auth.NewUserManager(db, []config.UserConfig{})
	am := auth.NewAuthManager(um)

	// Setup Router
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", am.LoginHandler)
	mux.HandleFunc("POST /login", am.LoginHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Dashboard"))
	})

	// Add CSRF Middleware
	handler := middleware.CSRFMiddleware(mux)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Client with Cookie Jar to store session cookies
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// 1. GET /login -> Should set session cookie and return CSRF token
	resp, err := client.Get(ts.URL + "/login")
	if err != nil {
		t.Fatalf("Failed to GET /login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check for session_token cookie
	cookies := jar.Cookies(resp.Request.URL)
	foundSession := false
	for _, c := range cookies {
		if c.Name == "session_token" {
			foundSession = true
			break
		}
	}
	if !foundSession {
		t.Error("Expected session_token cookie to be set on GET /login")
	}

	// Read body to find CSRF token
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)

	// The pattern looks for name="csrf_token" value="..."
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("Could not find CSRF token in login page")
	}
	csrfToken := matches[1]
	if csrfToken == "" {
		t.Error("CSRF token is empty")
	}
	t.Logf("Got CSRF Token: %s", csrfToken)

	// 2. POST /login WITHOUT CSRF token -> Should fail (403)
	// We need to use the SAME client to keep the session cookie
	form := url.Values{}
	form.Add("username", "admin")
	form.Add("password", "admin") // Default admin created by NewUserManager

	resp, err = client.PostForm(ts.URL + "/login", form)
	if err != nil {
		t.Fatalf("Failed to POST /login: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden when missing CSRF token, got %d", resp.StatusCode)
	}

	// 3. POST /login WITH CSRF token -> Should succeed (303 See Other -> /)
	form.Add("csrf_token", csrfToken)

	resp, err = client.PostForm(ts.URL + "/login", form)
	if err != nil {
		t.Fatalf("Failed to POST /login: %v", err)
	}
	defer resp.Body.Close()

	// If redirects followed, we should be at "/" with 200 OK
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK after login (redirected), got %d", resp.StatusCode)
	}

	// Verify we are at "/"
	if resp.Request.URL.Path != "/" {
		t.Errorf("Expected to be redirected to /, got %s", resp.Request.URL.Path)
	}
}
