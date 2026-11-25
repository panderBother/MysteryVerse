-- +goose Up

ALTER TABLE script_phases
    DROP CHECK ck_script_phases_type,
    ADD CONSTRAINT ck_script_phases_type CHECK (
        phase_type IN (
            'read',
            'opening',
            'search',
            'private_chat',
            'discussion',
            'final_statement',
            'vote',
            'settlement',
            'replay',
            'night'
        )
    );

-- +goose Down

ALTER TABLE script_phases
    DROP CHECK ck_script_phases_type,
    ADD CONSTRAINT ck_script_phases_type CHECK (
        phase_type IN ('read', 'search', 'discussion', 'vote', 'night', 'settlement')
    );
