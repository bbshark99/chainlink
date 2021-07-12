package migrations

import (
	"fmt"
	"math/big"
	"os"

	"gorm.io/gorm"
)

const up47 = `
CREATE TABLE evm_chains (
	id numeric(78,0) PRIMARY KEY,
	-- l1_chain_id numeric(78,0) REFERENCES evm_chains (id) -- This will come in a followup
)

CREATE TABLE nodes (
	id serial PRIMARY KEY,
	name varchar(255) NOT NULL CHECK (name != ''),
	chain_id numeric(78,0) NOT NULL REFERENCES evm_chains (id),
	ws_url string CHECK (ws_url != ''),
	http_url string CHECK (http_url != ''),
	send_only bool NOT NULL CONSTRAINT primary_or_sendonly CHECK (
		(send_only AND ws_url IS NULL AND http_url IS NOT NULL)
		OR
		(!send_only AND ws_url IS NOT NULL)
	)
)

CREATE INDEX idx_nodes_chain_id ON nodes (chain_id);
CREATE UNIQUE INDEX idx_nodes_unique_name ON nodes (lower(name));

ALTER TABLE eth_txes ADD COLUMN chain_id numeric(78,0) REFERENCES evm_chains (id);
ALTER TABLE log_broadcasts ADD COLUMN chain_id numeric(78,0) REFERENCES evm_chains (id);
ALTER TABLE heads ADD COLUMN chain_id numeric(78,0) REFERENCES evm_chains (id);

INSERT INTO evm_chains (id, l1_chain_id) VALUES (?, ?);
UPDATE eth_txes SET chain_id = ?;
UPDATE log_broadcasts SET chain_id = ?;
UPDATE heads SET chain_id = ?;

DROP INDEX idx_eth_txes_min_unconfirmed_nonce_for_key;
DROP INDEX idx_eth_txes_nonce_from_address;
DROP INDEX idx_only_one_in_progress_tx_per_account;
DROP INDEX idx_eth_txes_state_from_address;
DROP INDEX idx_eth_txes_unstarted_subject_id;
CREATE INDEX idx_eth_txes_min_unconfirmed_nonce_for_key_chain_id ON eth_txes(chain_id, from_address, nonce) WHERE state = 'unconfirmed'::eth_txes_state;
CREATE UNIQUE INDEX idx_eth_txes_nonce_from_address_per_chain_id ON eth_txes(chain_id, from_address, nonce);
CREATE UNIQUE INDEX idx_only_one_in_progress_tx_per_account_id_per_chain_id ON eth_txes(chain_id, from_address) WHERE state = 'in_progress'::eth_txes_state;
CREATE INDEX idx_eth_txes_state_from_address_chain_id ON eth_txes(chain_id, from_address, state) WHERE state <> 'confirmed'::eth_txes_state;
CREATE INDEX idx_eth_txes_unstarted_subject_id_chain_id ON eth_txes(chain_id, subject, id) WHERE subject IS NOT NULL AND state = 'unstarted'::eth_txes_state;

DROP INDEX idx_heads_hash;
DROP INDEX idx_heads_number;
CREATE UNIQUE INDEX idx_heads_chain_id_hash ON heads(chain_id, hash);
CREATE INDEX idx_heads_chain_id_number ON heads(chain_id, number);

ALTER TABLE eth_txes ALTER COLUMN chain_id SET NOT NULL;
ALTER TABLE log_broadcasts ALTER COLUMN chain_id SET NOT NULL;
ALTER TABLE heads ALTER COLUMN chain_id SET NOT NULL;
`

const down47 = `
`

func init() {
	chainIDStr := os.Getenv("ETH_CHAIN_ID")
	if chainIDStr == "" {
		chainIDStr = "1"
	}
	chainID, ok := new(big.Int).SetString(chainIDStr, 10)
	if !ok {
		panic(fmt.Sprintf("ETH_CHAIN_ID was invalid, expected a number, got: %s", chainIDStr))
	}
	Migrations = append(Migrations, &Migration{
		ID: "0047_multichain",
		Migrate: func(db *gorm.DB) error {
			return db.Exec(up47, chainID.String(), chainID.String()).Error
		},
		Rollback: func(db *gorm.DB) error {
			return db.Exec(down47).Error
		},
	})
}
