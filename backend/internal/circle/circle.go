// Package circle owns the trust circle domain (R-03): the user's contacts.
// The store underneath is sqlc-generated from the same DDL that ships to
// production (backend/sqlc.yaml reads the migrations directory).
package circle

import "github.com/tiagofur/imalive/backend/internal/circle/store"

// Contact is a member of a user's trust circle. DisplayName maps to the
// contract's `name` field (schema Contact in /contracts/openapi.yaml); the
// column is more explicit than the API field and the mapping lives here and
// in the handlers, nowhere else.
type Contact = store.Contact

// MaxContacts is the database-enforced ceiling (UNIQUE(user_id, position)
// with CHECK 1..5). The per-plan quota (R-13, week 3) is business logic on
// top; the schema guarantees the absolute limit even under app bugs.
const MaxContacts = 5
