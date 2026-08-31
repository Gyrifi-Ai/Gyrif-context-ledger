ALTER TABLE proposals ADD COLUMN change_ids TEXT NOT NULL DEFAULT '[]';

UPDATE proposals
SET change_ids = (
    SELECT json_group_array(change_id)
    FROM (
        SELECT change_id
        FROM proposal_changes
        WHERE proposal_id = proposals.id
        ORDER BY ordinal
    )
);
