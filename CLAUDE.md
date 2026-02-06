# CLAUDE.md - GoMonitoring Project

## Project Overview
GoMonitoring is a full-stack web-based infrastructure monitoring application for Linux and Windows servers via SSH. Built with Go backend + vanilla JavaScript/HTML/CSS frontend. Designed for air-gapped networks.

## Architecture
- **Backend**: Go 1.25.6, net/http (no framework), html/template, gorilla/websocket, SQLite3
- **Frontend**: Vanilla JS, CSS3 with CSS Variables, Chart.js, xterm.js
- **Auth**: Session cookies (24h), bcrypt passwords, CSRF tokens
- **Real-time**: WebSocket (5s refresh), 7-day historical data retention

## Key Directories
- `cmd/server/` - Main entry point (port :8080)
- `handlers/` - HTTP handlers (dashboard, machine, users, audit, etc.)
- `models/system.go` - Data structures (DashboardData, MachineDetailData, etc.)
- `templates/` - Go HTML templates (base layout + pages)
- `static/css/style.css` - Main stylesheet (~4000 lines)
- `static/js/` - Client-side JS (dashboard_refresh, machines, charts, etc.)
- `auth/` - Authentication & session management
- `config/` - YAML config parser
- `collectors/` - SSH metric collection (CPU, memory, disk, network)
- `storage/` - SQLite database layer

## Template System
- All pages extend `templates/layout/base.html`
- Template data structs must include: `Title`, `Status`, `Role`, `Username`, `CSRFToken`
- Custom template functions: `lower` (strings.ToLower), `upper` (strings.ToUpper)
- Template functions must be registered in ALL handlers that parse templates (dashboard.go, machine.go, pages.go, audit_page.go, users_page.go)

## Important Patterns
- All page handlers pass `Username` and `Role` from `auth.AuthManager`
- CSS uses CSS Custom Properties for theming (light/dark mode via `[data-theme="dark"]`)
- CSS version query param (`?v=12`) must be bumped on changes
- Files may have BOM characters - use `Write` tool instead of `Edit` for full file replacements
- The `sidebar-border` CSS variable is referenced but defined as `--border-color` in practice

## UI/UX Standards
- Icons: Inline SVG (feather-style, 20x20 for nav, 14-18 for buttons)
- No emoji in production templates - use SVG icons instead
- Toast notifications via `showToast(message, type, duration)` function in base.html
- All forms should have proper `autocomplete` attributes
- All interactive elements need `aria-label` for accessibility
- Password fields should have show/hide toggle
- French language UI labels

## Development Notes
- Go toolchain download may fail in air-gapped environments
- Use `GOTOOLCHAIN=local` if network unavailable
- No external CDN dependencies (fonts bundled locally)
