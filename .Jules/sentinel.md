## 2025-05-18 - Configuration Persistence vs Runtime State
**Vulnerability:** Passwords were saved in plain text because the runtime configuration struct (which needed decrypted passwords for SSH) was being marshaled directly to disk.
**Learning:** Using the same struct for runtime state and persistence creates a conflict when security requirements differ (e.g., encryption at rest vs plain text in memory).
**Prevention:** Implement a dedicated "Save" method that creates a deep copy of the configuration, applies necessary transformations (like encryption) to the copy, and then saves it, ensuring the runtime state remains untouched and secure. Always fail-closed on encryption errors.
