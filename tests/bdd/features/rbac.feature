Feature: Kontrol akses API (RBAC)
  Akses endpoint REST dibatasi autentikasi (JWT) + izin per-peran.
  Administrator mem-bypass pengecekan izin.

  Scenario: Tanpa token ditolak
    Given aplikasi API siap dengan admin ter-seed
    When klien mengakses "GET" "/api/v1/access/user" tanpa token
    Then status respons adalah 401

  Scenario: Admin mengakses daftar user
    Given aplikasi API siap dengan admin ter-seed
    When admin login
    And klien mengakses "GET" "/api/v1/access/user" dengan token
    Then status respons adalah 200

  Scenario: User tanpa izin ditolak
    Given aplikasi API siap dengan admin ter-seed
    And ada user biasa "plain@example.com" dengan password "password123"
    When user "plain@example.com" dengan password "password123" login
    And klien mengakses "GET" "/api/v1/access/user" dengan token
    Then status respons adalah 403
