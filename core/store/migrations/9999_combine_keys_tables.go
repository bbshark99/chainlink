package migrations

import (
	"gorm.io/gorm"
)

const up9999 = `
CREATE TABLE encrypted_key_rings(
    id SERIAL PRIMARY KEY,
    encrypted_keys jsonb,
    updated_at timestamptz NOT NULL
);

CREATE TABLE eth_key_states(
	id SERIAL PRIMARY KEY,
	address bytea UNIQUE NOT NULL,
	next_nonce bigint DEFAULT 0,
	is_funding boolean DEFAULT false NOT NULL,
	last_used timestamp with time zone,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	CONSTRAINT chk_address_length CHECK ((octet_length(address) = 20))
);

ALTER TABLE eth_txes DROP CONSTRAINT eth_txes_from_address_fkey;
ALTER TABLE eth_txes ADD CONSTRAINT eth_txes_from_address_fkey FOREIGN KEY (from_address) REFERENCES eth_key_states(address);
`
const down9999 = `
DROP TABLE encrypted_key_rings;
DROP TABLE eth_key_states;
ALTER TABLE eth_txes DROP CONSTRAINT eth_txes_from_address_fkey;
ALTER TABLE eth_txes ADD CONSTRAINT eth_txes_from_address_fkey FOREIGN KEY (from_address) REFERENCES keys(address);
`

func init() {
	Migrations = append(Migrations, &Migration{
		ID: "9999_combine_keys_tables",
		Migrate: func(db *gorm.DB) error {
			return db.Exec(up9999).Error
		},
		Rollback: func(db *gorm.DB) error {
			return db.Exec(down9999).Error
		},
	})
}
