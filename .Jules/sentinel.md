## 2026-02-13 - [Hardcoded Origin Whitelist Vulnerability]
**Vulnerability:** A hardcoded `localhost` whitelist in `websocket.Upgrader.CheckOrigin` blocked legitimate production domains and created a false sense of security while breaking flexibility.
**Learning:** Hardcoded security whitelists are brittle and often lead to developers bypassing them entirely (e.g., `return true`) to make the app work, which destroys security.
**Prevention:** Use dynamic, logic-based checks (e.g., `Origin.Host == Request.Host`) instead of static lists. This aligns with standard security defaults while maintaining usability.
