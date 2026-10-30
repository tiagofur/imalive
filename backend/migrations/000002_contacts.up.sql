-- 000002_contacts.up.sql
-- contacts: el círculo de confianza (R-03). PII mínima por contacto:
--   display_name + phone (E.164). El contacto jamás crea cuenta ni token:
--   no hay FK hacia ninguna tabla de identidad además de user_id.
-- UNIQUE (user_id, phone): un contacto no se repite por usuario.
-- UNIQUE (user_id, position) + CHECK 1..5: techo duro de 5 contactos
--   a nivel DB (ContactList.maxItems del contrato); la cuota por plan
--   (R-13, semana 3) sigue siendo lógica de app.
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

CREATE TABLE contacts (
    id           uuid PRIMARY KEY,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    display_name text        NOT NULL
        CHECK (char_length(display_name) BETWEEN 1 AND 80),
    phone        text        NOT NULL
        CHECK (phone ~ '^\+[1-9][0-9]{7,14}$'),
    position     smallint    NOT NULL
        CHECK (position BETWEEN 1 AND 5),
    created_at   timestamptz NOT NULL DEFAULT now(),

    UNIQUE (user_id, phone),
    UNIQUE (user_id, position)
);

CREATE INDEX contacts_user_id_idx ON contacts (user_id);
