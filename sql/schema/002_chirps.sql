-- +goose Up
CREATE TABLE chirps(
	id UUID PRIMARY KEY,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	body TEXT NOT NULL,
	user_id UUID NOT NULL,

	CONSTRAINT fk_author
	FOREIGN KEY (user_id)
	REFERENCES users(id)
	ON DELETE CASCADE
);

-- +goose Down
DELETE FROM chirps;
