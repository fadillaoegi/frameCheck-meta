-- Provider token usage per run, when the provider reports it (0 otherwise).
ALTER TABLE runs
    ADD COLUMN input_tokens  bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    ADD COLUMN output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0);
